package inet256ipv4

import (
	"github.com/inet256/inet256/pkg/inet256"
)

type VPNConfig struct {
	LocalIP IPv4         `yaml:"local_ip"`
	Subnet  string       `yaml:"subnet"`
	Hosts   []HostConfig `yaml:"hosts"`
}

func DefaultVPNConfig() *VPNConfig {
	conf := &VPNConfig{
		LocalIP: IPv4FromString("192.168.250.100"),
		Subnet:  "192.168.250.0/24",
	}
	return conf
}

type HostConfig struct {
	INet256 inet256.Addr `yaml:"inet256"`
	IP      IPv4         `yaml:"ip"`
}
