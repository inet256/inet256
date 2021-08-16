package inet256d

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/brendoncarroll/go-p2p/s/udpswarm"
	"github.com/brendoncarroll/go-p2p/s/upnpswarm"
	"github.com/inet256/inet256/pkg/autopeering"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/discovery/celldisco"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const DefaultAPIAddr = "127.0.0.1:2560"

type PeerSpec struct {
	ID    inet256.Addr `yaml:"id"`
	Addrs []string     `yaml:"addrs"`
}

type TransportSpec struct {
	UDP      *UDPTransportSpec      `yaml:"udp,omitempty"`
	Ethernet *EthernetTransportSpec `yaml:"ethernet,omitempty"`

	NATUPnP bool `yaml:"nat_upnp"`
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
	APIAddr        string          `yaml:"api_addr"`
	Networks       []string        `yaml:"networks"`
	Transports     []TransportSpec `yaml:"transports"`
	Peers          []PeerSpec      `yaml:"peers"`

	Discovery   []DiscoverySpec   `yaml:"discovery"`
	AutoPeering []AutoPeeringSpec `yaml:"autopeering"`
}

func (c Config) GetAPIAddr() string {
	if c.APIAddr == "" {
		return DefaultAPIAddr
	}
	return c.APIAddr
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
	swarms := map[string]p2p.SecureSwarm{}
	for _, tspec := range c.Transports {
		sw, swname, err := makeTransport(tspec, privateKey)
		if err != nil {
			return nil, err
		}
		if tspec.NATUPnP {
			sw = upnpswarm.WrapSwarm(sw)
		}
		secSw, err := quicswarm.New(sw, privateKey)
		if err != nil {
			return nil, err
		}
		swname = "quic+" + swname
		swarms[swname] = secSw
	}
	addrSchema := multiswarm.NewSchemaFromSecureSwarms(swarms)
	// peers
	peers := inet256srv.NewPeerStore()
	for _, pspec := range c.Peers {
		addrs, err := serde.ParseAddrs(addrSchema.ParseAddr, pspec.Addrs)
		if err != nil {
			return nil, err
		}
		peers.Add(pspec.ID)
		peers.SetAddrs(pspec.ID, addrs)
	}
	// networks
	nspecs := []inet256.NetworkSpec{}
	for _, netName := range c.Networks {
		nspec, exists := networkNames[netName]
		if !exists {
			return nil, errors.Errorf("no network: %s", netName)
		}
		nspecs = append(nspecs, nspec)
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
			Networks:   nspecs,
			Peers:      peers,
		},
		DiscoveryServices:   dscSrvs,
		AutoPeeringServices: apSrvs,
		APIAddr:             c.APIAddr,
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
		Networks: DefaultNetworks(),
		APIAddr:  DefaultAPIAddr,
		Transports: []TransportSpec{
			{
				UDP: (*UDPTransportSpec)(strPtr("0.0.0.0:0")),
			},
		},
	}
}

func DefaultNetworks() []string {
	var names []string
	for name := range networkNames {
		names = append(names, name)
	}
	return names
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

func strPtr(x string) *string {
	return &x
}
