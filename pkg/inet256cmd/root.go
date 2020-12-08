package inet256cmd

import (
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/spf13/cobra"
)

func Execute() error {
	return rootCmd.Execute()
}

func Register(name string, index int, factory inet256.NetworkFactory) {
	netSpec := inet256.NetworkSpec{
		Name:    name,
		Index:   index,
		Factory: factory,
	}
	if _, exists := networks[netSpec.Index]; exists {
		panic("network by that name already exists")
	}
	networks[netSpec.Index] = netSpec
}

var (
	configPath string
	networks   = map[int]inet256.NetworkSpec{}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "--config=./path/to/config/yaml")
}

var rootCmd = &cobra.Command{
	Use:   "inet256",
	Short: "inet256: A secure network with a 256 bit address space",
}
