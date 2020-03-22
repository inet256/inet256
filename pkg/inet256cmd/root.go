package inet256cmd

import (
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/spf13/cobra"
)

func Execute() error {
	return rootCmd.Execute()
}

func Register(name string, factory inet256.NetworkFactory) {
	netSpec := inet256.NetworkSpec{
		Name:    name,
		Factory: factory,
	}
	if _, exists := networks[netSpec.Name]; exists {
		panic("network by that name already exists")
	}
	networks[netSpec.Name] = netSpec
}

var (
	configPath string
	networks   = map[string]inet256.NetworkSpec{}

	node *inet256.Node
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "--config=./path/to/config/yaml")
}

var rootCmd = &cobra.Command{
	Use:   "inet256",
	Short: "inet256: A secure network with a 256 bit address space",
}
