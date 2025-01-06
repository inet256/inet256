package inet256cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"go.inet256.org/inet256/pkg/inet256d"
)

func newStatusCmd(newClient func() (inet256d.AdminClient, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "prints status of the main node",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			res, err := client.GetStatus(ctx)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "MAIN ADDR: %v\n", res.MainAddr)
			fmt.Fprintf(w, "TRANSPORTS:\n")
			for _, addr := range res.TransportAddrs {
				fmt.Fprintf(w, "\t%s\n", addr)
			}
			fmt.Fprintf(w, "PEERS:\n")
			for _, status := range res.Peers {
				fmt.Fprintf(w, "\t%s\n", status.Addr)
				for addr, lastSeen := range status.LastSeen {
					fmt.Fprintf(w, "\t\t%s\t%v\n", addr, lastSeen)
				}
			}
			return nil
		},
	}
}
