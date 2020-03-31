package inet256cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/d/celltracker"
	"github.com/brendoncarroll/go-p2p/s/dtlsswarm"
	"github.com/brendoncarroll/go-p2p/s/natswarm"
	"github.com/brendoncarroll/go-p2p/s/udpswarm"
	"github.com/inet256/inet256/pkg/inet256"
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
	Networks       []string        `yaml:"networks"`
	Transports     []TransportSpec `yaml:"transports"`
	Peers          []PeerSpec      `yaml:"peers"`

<<<<<<< HEAD
	Discovery []DiscoverySpec `yaml:"discovery"`
}

func BuildParams(c *Config) (*inet256.Params, error) {
=======
	CellTrackers []string   `yaml:"cell_trackers"`
	Peers        []PeerSpec `yaml:"peers"`

	configPath string
}

func (c *Config) BuildParams() (*inet256.Params, error) {
	keyPath := c.PrivateKeyPath
	if c.configPath != "" {
		keyPath = filepath.Join(filepath.Dir(c.configPath), c.PrivateKeyPath)
	}

>>>>>>> 74e8d87 (wip)
	// private key
	keyPEMData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(keyPEMData)
	if block == nil {
		return nil, errors.New("key file does not contain PEM")
	}
	if block.Type != "PRIVATE KEY" {
		return nil, errors.New("wrong type for PEM block")
	}
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	privateKey2, ok := privateKey.(p2p.PrivateKey)
	if !ok {
		return nil, errors.New("unknown private key")
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
		secSw := dtlsswarm.New(sw, privateKey2)
		swarms["dtls+"+tspec.Type] = secSw
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
		PrivateKey: privateKey2,
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
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	privKeyData, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyData,
	})
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
	c.configPath = p
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
