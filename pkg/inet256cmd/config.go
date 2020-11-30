package inet256cmd

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/d/celltracker"
	"github.com/brendoncarroll/go-p2p/s/natswarm"
	"github.com/brendoncarroll/go-p2p/s/udpswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mocksecswarm"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var swarmFactories = map[string]func(addr string) (p2p.Swarm, error){
	"udp": func(addr string) (p2p.Swarm, error) {
		usw, err := udpswarm.New(addr)
		return usw, err
	},
}

type PeerSpec struct {
	ID    p2p.PeerID `yaml:"id"`
	Addrs []string   `yaml:"addrs"`
}

type TransportSpec struct {
	Type      string `yaml:"type"`
	Addr      string `yaml:"addr"`
	BehindNAT bool   `yaml:"behind_nat"`
}

type DiscoverySpec struct {
	CellTracker *CellTrackerSpec `yaml:"cell_tracker"`
}

type CellTrackerSpec = string

type Config struct {
	PrivateKeyPath string          `yaml:"private_key_path"`
	APIAddr        string          `yaml:"api_addr"`
	Networks       []string        `yaml:"networks"`
	Transports     []TransportSpec `yaml:"transports"`
	Peers          []PeerSpec      `yaml:"peers"`

	Discovery []DiscoverySpec `yaml:"discovery"`
}

func (c Config) GetAPIAddr() string {
	if c.APIAddr == "" {
		return defaultAPIAddr
	}
	return c.APIAddr
}

func BuildParams(configPath string, c *Config) (*inet256.Params, error) {
	keyPath := c.PrivateKeyPath
	if strings.HasPrefix(c.PrivateKeyPath, "./") {
		keyPath = filepath.Join(filepath.Dir(configPath), c.PrivateKeyPath)
	}
	// private key
	keyPEMData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	privateKey, err := inet256.ParsePrivateKeyPEM(keyPEMData)
	if err != nil {
		return nil, err
	}
	swarms := map[string]p2p.SecureSwarm{}
	for _, tspec := range c.Transports {
		factory, exists := swarmFactories[tspec.Type]
		if !exists {
			return nil, fmt.Errorf("unrecognized transport type %s: ", tspec.Type)
		}

		sw, err := factory(tspec.Addr)
		if err != nil {
			return nil, err
		}
		if tspec.BehindNAT {
			sw = natswarm.WrapSwarm(sw)
		}
		secSw := mocksecswarm.New(sw, privateKey)
		swarms["mocksec+"+tspec.Type] = secSw
	}

	// peers
	peers := inet256.NewPeerStore()
	for _, pspec := range c.Peers {
		peers.AddPeer(pspec.ID)
		for _, addrStr := range pspec.Addrs {
			peers.AddAddr(pspec.ID, addrStr)
		}
	}

	nspecs := []inet256.NetworkSpec{}
	for _, nspec := range networks {
		nspecs = append(nspecs, nspec)
	}

	params := &inet256.Params{
		PrivateKey: privateKey,
		Swarms:     swarms,
		Networks:   nspecs,
		Peers:      peers,
	}

	return params, nil
}

func DefaultConfig() Config {
	return Config{
		Networks: []string{"onehop"},
		Transports: []TransportSpec{
			{
				Type: "udp",
				Addr: "0.0.0.0:",
			},
		},
	}
}

func SaveDefaultConfig(p string) error {
	privateKey, err := generateKey()
	if err != nil {
		return err
	}
	privKeyPEM, err := inet256.MarshalPrivateKeyPEM(privateKey)
	if err != nil {
		return err
	}
	dirPath := filepath.Dir(p)
	keyPath := filepath.Join(dirPath, "inet256_private.pem")
	if err := ioutil.WriteFile(keyPath, privKeyPEM, 0644); err != nil {
		return err
	}
	nspecs := []inet256.NetworkSpec{}
	for _, nspec := range networks {
		nspecs = append(nspecs, nspec)
	}
	c := DefaultConfig()
	c.PrivateKeyPath = keyPath
	return SaveConfig(p, c)
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

func SaveConfig(p string, c Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(p, data, 0644)
}

func setupDiscovery(spec DiscoverySpec) (p2p.DiscoveryService, error) {
	switch {
	case spec.CellTracker != nil:
		return celltracker.NewClient(*spec.CellTracker)
	default:
		return nil, errors.Errorf("empty spec")
	}
}
