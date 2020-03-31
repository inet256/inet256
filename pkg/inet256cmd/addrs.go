package inet256cmd

import (
	"sort"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(localAddrsCmd)
}

var localAddrsCmd = &cobra.Command{
	Use:   "local-addrs",
	Short: "lists the local addresses",
	RunE: func(cmd *cobra.Command, args []string) error {
		node, _, err := setupNode(cmd, args)
		if err != nil {
			return err
		}
		addrs := node.TransportAddrs()
		sort.Strings(addrs)
		for _, a := range addrs {
			cmd.Println(a)
		}
		return nil
	},
}
