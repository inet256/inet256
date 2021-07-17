package inet256ipv6

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/serde"
	"gopkg.in/yaml.v3"
)

type PortalConfig struct {
	PrivateKey string         `yaml:"private_key"`
	Allowed    []inet256.Addr `yaml:"allowed"`
}

func (c *PortalConfig) GetPrivateKey() (p2p.PrivateKey, error) {
	data, err := base64.StdEncoding.DecodeString(c.PrivateKey)
	if err != nil {
		return nil, err
	}
	return serde.ParsePrivateKey(data)
}

func (c *PortalConfig) GetAllowFunc() AllowFunc {
	set := make(map[inet256.Addr]struct{}, len(c.Allowed))
	for _, a := range c.Allowed {
		set[a] = struct{}{}
	}
	return func(x inet256.Addr) bool {
		_, exists := set[x]
		return exists
	}
}

func DefaultPortalConfig() PortalConfig {
	_, privKey, err := MineAddr(context.Background(), rand.Reader, WorthItBits)
	if err != nil {
		panic(err)
	}
	pkBytes, err := serde.MarshalPrivateKey(privKey)
	if err != nil {
		panic(err)
	}
	pkb64str := base64.StdEncoding.EncodeToString(pkBytes)
	return PortalConfig{
		PrivateKey: pkb64str,
	}
}

func ParsePortalConfig(data []byte) (*PortalConfig, error) {
	var config PortalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
