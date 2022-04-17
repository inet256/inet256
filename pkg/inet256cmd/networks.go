package inet256cmd

import (
	"bufio"
	"fmt"

	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/spf13/cobra"
)

func newNetworksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-networks",
		Short: "generates a private key and writes it to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			w := bufio.NewWriter(cmd.OutOrStdout())
			for _, n := range inet256d.ListNetworks() {
				fmt.Fprintf(w, "%s\n", n)
			}
			return w.Flush()
		},
	}
}
