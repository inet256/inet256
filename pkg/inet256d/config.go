package inet256d

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/udpswarm"
	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/networks/beaconnet"
	"github.com/inet256/inet256/networks/floodnet"
	"github.com/inet256/inet256/networks/multinet"
	"github.com/inet256/inet256/pkg/autopeering"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/discovery/celldisco"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const DefaultAPIEndpoint = "127.0.0.1:2560"

type PeerSpec struct {
	ID    inet256.Addr `yaml:"id"`
	Addrs []string     `yaml:"addrs"`
}

type NetworkSpec struct {
	FloodNet  *struct{}              `yaml:"floodnet,omitempty"`
	BeaconNet *struct{}              `yaml:"beaconnet,omitempty"`
	OneHop    *struct{}              `yaml:"onehop,omitempty"`
	Multi     map[string]NetworkSpec `yaml:"multi,omitempty"`
}

type TransportSpec struct {
	UDP      *UDPTransportSpec      `yaml:"udp,omitempty"`
	Ethernet *EthernetTransportSpec `yaml:"ethernet,omitempty"`
}

type UDPTransportSpec string
type EthernetTransportSpec string

type DiscoverySpec struct {
	Cell  *CellDiscoverySpec `yaml:"cell"`
	Local *LocalDiscoverySpec
}

type CellDiscoverySpec = string

type LocalDiscoverySpec struct {
	MulticastAddr string `yaml:"multicast_addr"`
}

type AutoPeeringSpec struct {
}

type Config struct {
	PrivateKeyPath string          `yaml:"private_key_path"`
	APIEndpoint    string          `yaml:"api_endpoint"`
	Network        NetworkSpec     `yaml:"network"`
	Transports     []TransportSpec `yaml:"transports"`
	Peers          []PeerSpec      `yaml:"peers"`

	Discovery   []DiscoverySpec   `yaml:"discovery"`
	AutoPeering []AutoPeeringSpec `yaml:"autopeering"`
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
	keyPEMData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	privateKey, err := serde.ParsePrivateKeyPEM(keyPEMData)
	if err != nil {
		return nil, err
	}
	// transports
	swarms := map[string]p2p.Swarm{}
	for _, tspec := range c.Transports {
		sw, swname, err := makeTransport(tspec, privateKey)
		if err != nil {
			return nil, err
		}
		swarms[swname] = sw
	}
	// peers
	addrSchema := inet256srv.NewAddrSchema(swarms)
	peers := inet256srv.NewPeerStore()
	for _, pspec := range c.Peers {
		addrs, err := serde.ParseAddrs(addrSchema.ParseAddr, pspec.Addrs)
		if err != nil {
			return nil, err
		}
		peers.Add(pspec.ID)
		peers.SetAddrs(pspec.ID, addrs)
	}
	// network
	networkFactory, err := networkFactoryFromSpec(c.Network)
	if err != nil {
		return nil, err
	}
	// discovery
	dscSrvs := []discovery.Service{}
	for _, spec := range c.Discovery {
		dscSrv, err := makeDiscoveryService(spec, addrSchema)
		if err != nil {
			return nil, err
		}
		dscSrvs = append(dscSrvs, dscSrv)
	}
	// autopeering
	apSrvs := []autopeering.Service{}
	for _, spec := range c.AutoPeering {
		apSrv, err := makeAutoPeeringService(spec, addrSchema)
		if err != nil {
			return nil, err
		}
		apSrvs = append(apSrvs, apSrv)
	}

	params := &Params{
		MainNodeParams: inet256srv.Params{
			PrivateKey: privateKey,
			Swarms:     swarms,
			NewNetwork: networkFactory,
			Peers:      peers,
		},
		DiscoveryServices:   dscSrvs,
		AutoPeeringServices: apSrvs,
		APIAddr:             c.APIEndpoint,
	}
	return params, nil
}

func makeTransport(spec TransportSpec, privKey p2p.PrivateKey) (p2p.Swarm, string, error) {
	switch {
	case spec.UDP != nil:
		s, err := udpswarm.New(string(*spec.UDP))
		if err != nil {
			return nil, "", err
		}
		return s, "udp", nil
	case spec.Ethernet != nil:
		return nil, "", errors.Errorf("ethernet transport not implemented")
	default:
		return nil, "", errors.Errorf("empty transport spec")
	}
}

func makeDiscoveryService(spec DiscoverySpec, addrSchema multiswarm.AddrSchema) (discovery.Service, error) {
	switch {
	case spec.Cell != nil:
		return celldisco.New(*spec.Cell)
	default:
		return nil, errors.Errorf("empty discovery spec")
	}
}

func makeAutoPeeringService(spec AutoPeeringSpec, addrSchema multiswarm.AddrSchema) (autopeering.Service, error) {
	switch {
	default:
		return nil, errors.Errorf("empty autopeering spec")
	}
}

func DefaultConfig() Config {
	return Config{
		Network:     DefaultNetwork(),
		APIEndpoint: DefaultAPIEndpoint,
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
	data, err := ioutil.ReadFile(p)
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
	return ioutil.WriteFile(p, data, 0o644)
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

func networkFactoryFromSpec(spec NetworkSpec) (networks.Factory, error) {
	switch {
	case spec.BeaconNet != nil:
		return beaconnet.Factory, nil
	case spec.FloodNet != nil:
		return floodnet.Factory, nil
	case spec.OneHop != nil:
		return inet256srv.OneHopFactory, nil
	case spec.Multi != nil:
		netFacts := make(map[multinet.NetworkCode]networks.Factory)
		for codeStr, spec := range spec.Multi {
			code, err := codeFromString(codeStr)
			if err != nil {
				return nil, err
			}
			factory, err := networkFactoryFromSpec(spec)
			if err != nil {
				return nil, err
			}
			netFacts[code] = factory
		}
		return multinet.NewFactory(netFacts), nil
	default:
		return nil, errors.Errorf("empty network spec")
	}
}

func codeFromString(x string) (multinet.NetworkCode, error) {
	if len(x) > 8 {
		return [8]byte{}, errors.Errorf("network code %q is too long, must be <= 8 bytes", x)
	}
	b := []byte(x)
	for len(b) < 8 {
		b = append(b, 0x00)
	}
	return *(*[8]byte)(b), nil
}
