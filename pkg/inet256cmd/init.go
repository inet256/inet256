package inet256cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init creates a config file and private key",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configPath == "" {
			return errors.New("config path must be provided")
		}
		if _, err := os.Stat(configPath); err == nil {
			return errors.New("already entry at config path")
		}
		return SaveDefaultConfig(configPath)
	},
}
