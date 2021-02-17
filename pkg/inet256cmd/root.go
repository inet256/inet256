package inet256cmd

import (
	"github.com/spf13/cobra"
)

func Execute() error {
	return rootCmd.Execute()
}

var (
	configPath string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "--config=./path/to/config/yaml")
}

var rootCmd = &cobra.Command{
	Use:   "inet256",
	Short: "inet256: A secure network with a 256 bit address space",
}
