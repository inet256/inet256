package inet256d

import (
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/udpswarm"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"

	"github.com/inet256/inet256/internal/slices2"
	"github.com/inet256/inet256/networks/beaconnet"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/discovery/centraldisco"
	"github.com/inet256/inet256/pkg/discovery/landisco"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/peers"
	"github.com/inet256/inet256/pkg/serde"
)

const DefaultAPIEndpoint = "unix:///run/inet256.sock"

type PeerSpec struct {
	ID    inet256.Addr `yaml:"id"`
	Addrs []string     `yaml:"addrs"`
}

type NetworkSpec struct {
	BeaconNet *struct{} `yaml:"beaconnet,omitempty"`
	OneHop    *struct{} `yaml:"onehop,omitempty"`
}

type TransportSpec struct {
	UDP      *UDPTransportSpec      `yaml:"udp,omitempty"`
	Ethernet *EthernetTransportSpec `yaml:"ethernet,omitempty"`
}

type UDPTransportSpec string
type EthernetTransportSpec string

type DiscoverySpec struct {
	AutoPeering bool `yaml:"autopeering"`

	Local   *LocalDiscoverySpec   `yaml:"local,omitempty"`
	Central *CentralDiscoverySpec `yaml:"central,omitempty"`
}

type LocalDiscoverySpec struct {
	Interfaces []string `yaml:"interfaces"`

	AnnouncePeriod time.Duration `yaml:"announce_period,omitempty"`
}

func (s LocalDiscoverySpec) GetAnnouncePeriod() time.Duration {
	const DefaultAnnouncePeriod = 15 * time.Second
	if s.AnnouncePeriod > 0 {
		return s.AnnouncePeriod

	}
	return DefaultAnnouncePeriod
}

type CentralDiscoverySpec struct {
	Endpoint string        `yaml:"endpoint"`
	Period   time.Duration `yaml:"period,omitempty"`
}

func (s *CentralDiscoverySpec) GetPeriod() time.Duration {
	const DefaultPeriod = 15 * time.Second
	if s.Period > 0 {
		return s.Period
	}
	return DefaultPeriod
}

type Config struct {
	PrivateKeyPath string          `yaml:"private_key_path"`
	APIEndpoint    string          `yaml:"api_endpoint"`
	Network        NetworkSpec     `yaml:"network"`
	Transports     []TransportSpec `yaml:"transports"`
	Peers          []PeerSpec      `yaml:"peers"`

	Discovery []DiscoverySpec `yaml:"discovery"`
}

func (c Config) GetAPIAddr() string {
	if c.APIEndpoint == "" {
		return DefaultAPIEndpoint
	}
	return c.APIEndpoint
}

func MakeParams(configPath string, c Config) (*Params, error) {
	// private key
	keyPath := c.PrivateKeyPath
	if strings.HasPrefix(c.PrivateKeyPath, "./") {
		keyPath = filepath.Join(filepath.Dir(configPath), c.PrivateKeyPath)
	}
	keyPEMData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	privateKey, err := serde.ParsePrivateKeyPEM(keyPEMData)
	if err != nil {
		return nil, err
	}
	// transports
	swarms := map[string]multiswarm.DynSwarm{}
	for _, tspec := range c.Transports {
		sw, swname, err := makeTransport(tspec, privateKey)
		if err != nil {
			return nil, err
		}
		swarms[swname] = sw
	}
	// peers
	addrSchema := mesh256.NewAddrSchema(swarms)
	ps := mesh256.NewPeerStore()
	for _, pspec := range c.Peers {
		addrs, err := serde.ParseAddrs(addrSchema.ParseAddr, pspec.Addrs)
		if err != nil {
			return nil, err
		}
		ps.Add(pspec.ID)
		peers.SetAddrs[multiswarm.Addr](ps, pspec.ID, addrs)
	}
	// network
	networkFactory, err := networkFactoryFromSpec(c.Network)
	if err != nil {
		return nil, err
	}
	// discovery
	dscSrvs := []discovery.Service{}
	for _, spec := range c.Discovery {
		dscSrv, err := makeDiscovery(spec, addrSchema)
		if err != nil {
			return nil, err
		}
		dscSrvs = append(dscSrvs, dscSrv)
	}

	params := &Params{
		MainNodeParams: mesh256.Params{
			PrivateKey: privateKey,
			Swarms:     swarms,
			NewNetwork: networkFactory,
			Peers:      ps,
		},
		Discovery:           dscSrvs,
		APIAddr:             c.APIEndpoint,
		TransportAddrParser: addrSchema.ParseAddr,
	}
	return params, nil
}

func makeTransport(spec TransportSpec, privKey inet256.PrivateKey) (multiswarm.DynSwarm, string, error) {
	switch {
	case spec.UDP != nil:
		s, err := udpswarm.New(string(*spec.UDP))
		if err != nil {
			return nil, "", err
		}
		return multiswarm.WrapSwarm[udpswarm.Addr](s), "udp", nil
	case spec.Ethernet != nil:
		return nil, "", errors.Errorf("ethernet transport not implemented")
	default:
		return nil, "", errors.Errorf("empty transport spec")
	}
}

func makeDiscovery(spec DiscoverySpec, addrSchema multiswarm.AddrSchema) (discovery.Service, error) {
	switch {
	case spec.Local != nil:
		return &landisco.Service{
			Interfaces:     spec.Local.Interfaces,
			AnnouncePeriod: spec.Local.GetAnnouncePeriod(),
		}, nil
	case spec.Central != nil:
		endpoint := spec.Central.Endpoint
		var opts []grpc.DialOption
		if strings.HasPrefix(endpoint, "http://") {
			endpoint = strings.TrimPrefix(endpoint, "http://")
			opts = append(opts, grpc.WithInsecure())
		}
		endpoint = strings.TrimPrefix(endpoint, "https://")
		gc, err := grpc.Dial(endpoint, opts...)
		if err != nil {
			return nil, err
		}
		client := centraldisco.NewClient(gc)
		return centraldisco.NewService(client, spec.Central.GetPeriod()), nil
	default:
		return nil, errors.Errorf("empty discovery spec")
	}
}

func DefaultConfig() Config {
	return Config{
		PrivateKeyPath: "./private_key.pem",
		Network:        DefaultNetwork(),
		APIEndpoint:    DefaultAPIEndpoint,
		Transports: []TransportSpec{
			{
				UDP: (*UDPTransportSpec)(strPtr("0.0.0.0:0")),
			},
		},
	}
}

func DefaultNetwork() NetworkSpec {
	return NetworkSpec{
		BeaconNet: &struct{}{},
	}
}

func LoadConfig(p string) (*Config, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	return c, nil
}

func SaveConfig(config Config, p string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func ListNetworks() (ret []string) {
	ty := reflect.TypeOf(NetworkSpec{})
	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i)
		name := strings.ToLower(field.Name)
		ret = append(ret, name)
	}
	sort.Strings(ret)
	return ret
}

func strPtr(x string) *string {
	return &x
}

func networkFactoryFromSpec(spec NetworkSpec) (mesh256.NetworkFactory, error) {
	switch {
	case spec.BeaconNet != nil:
		return beaconnet.Factory, nil
	default:
		return nil, errors.Errorf("empty network spec")
	}
}

func InterfaceNames() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	return slices2.Map(ifaces, func(x net.Interface) string { return x.Name }), nil
}
