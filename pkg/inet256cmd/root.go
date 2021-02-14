package inet256cmd

import (
	"encoding/binary"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/spf13/cobra"
)

func Execute() error {
	return rootCmd.Execute()
}

func IndexFromString(x string) uint64 {
	if len(x) > 8 {
		panic("string to long")
	}
	b := []byte(x)
	for len(b) < 8 {
		b = append(b, 0x00)
	}
	return binary.LittleEndian.Uint64(b)
}

func Register(index uint64, name string, factory inet256.NetworkFactory) {
	netSpec := inet256.NetworkSpec{
		Name:    name,
		Index:   index,
		Factory: factory,
	}
	if _, exists := networkIndexes[netSpec.Index]; exists {
		panic("network by that name already exists")
	}
	if _, exists := networkNames[netSpec.Name]; exists {
		panic("network already exists at that index")
	}
	networkIndexes[netSpec.Index] = netSpec
	networkNames[netSpec.Name] = netSpec
}

var (
	configPath     string
	networkIndexes = map[uint64]inet256.NetworkSpec{}
	networkNames   = map[string]inet256.NetworkSpec{}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "--config=./path/to/config/yaml")
}

var rootCmd = &cobra.Command{
	Use:   "inet256",
	Short: "inet256: A secure network with a 256 bit address space",
}
