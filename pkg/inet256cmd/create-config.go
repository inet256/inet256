package inet256cmd

import (
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(createConfigCmd)
}

var createConfigCmd = &cobra.Command{
	Use:   "create-config",
	Short: "creates a new default config and writes it to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := DefaultConfig()
		data, err := yaml.Marshal(c)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		out.Write(data)
		return nil
	},
}
