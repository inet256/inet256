package inet256d

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/noiseswarm"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/brendoncarroll/go-p2p/s/udpswarm"
	"github.com/brendoncarroll/go-p2p/s/upnpswarm"
	"github.com/inet256/inet256/pkg/autopeering"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/discovery/celldisco"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const DefaultAPIAddr = "127.0.0.1:25600"

type PeerSpec struct {
	ID    p2p.PeerID `yaml:"id"`
	Addrs []string   `yaml:"addrs"`
}

type TransportSpec struct {
	QUIC *QUICTransportSpec `yaml:"quic,omitempty"`

	UDP      *UDPTransportSpec      `yaml:"udp,omitempty"`
	Ethernet *EthernetTransportSpec `yaml:"ethernet,omitempty"`

	NATUPnP bool `yaml:"nat_upnp"`
}

type UDPTransportSpec string
type QUICTransportSpec string
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
	privateKey, err := inet256.ParsePrivateKeyPEM(keyPEMData)
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
		secSw, ok := sw.(p2p.SecureSwarm)
		if !ok {
			secSw = noiseswarm.New(sw, privateKey)
			swname = "noise+" + swname
		}
		swarms[swname] = secSw
	}
	// peers
	peers := inet256.NewPeerStore()
	for _, pspec := range c.Peers {
		peers.Add(pspec.ID)
		peers.SetAddrs(pspec.ID, pspec.Addrs)
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
		dscSrv, err := makeDiscoveryService(spec)
		if err != nil {
			return nil, err
		}
		dscSrvs = append(dscSrvs, dscSrv)
	}
	// autopeering
	apSrvs := []autopeering.Service{}
	for _, spec := range c.AutoPeering {
		apSrv, err := makeAutoPeeringService(spec)
		if err != nil {
			return nil, err
		}
		apSrvs = append(apSrvs, apSrv)
	}

	params := &Params{
		MainNodeParams: inet256.Params{
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
	case spec.QUIC != nil:
		s, err := quicswarm.New(string(*spec.QUIC), privKey)
		if err != nil {
			return nil, "", err
		}
		return s, "quic", err
	case spec.Ethernet != nil:
		return nil, "", errors.Errorf("ethernet transport not implemented")
	default:
		return nil, "", errors.Errorf("empty transport spec")
	}
}

func makeDiscoveryService(spec DiscoverySpec) (discovery.Service, error) {
	switch {
	case spec.Cell != nil:
		return celldisco.New(*spec.Cell)
	default:
		return nil, errors.Errorf("empty discovery spec")
	}
}

func makeAutoPeeringService(spec AutoPeeringSpec) (autopeering.Service, error) {
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
