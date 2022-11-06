package inet256cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/inet256/inet256/pkg/inet256d"
)

func newStatusCmd(newClient func() (inet256d.AdminClient, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "prints status of the main node",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.GetStatus(ctx, &emptypb.Empty{})
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "LOCAL ADDR: %v\n", res.LocalAddr)
			fmt.Fprintf(w, "TRANSPORTS:\n")
			for _, addr := range res.TransportAddrs {
				fmt.Fprintf(w, "\t%s\n", addr)
			}
			fmt.Fprintf(w, "PEERS:\n")
			for _, status := range res.PeerStatus {
				fmt.Fprintf(w, "\t%s\n", status.Addr)
				for addr, lastSeen := range status.LastSeen {
					fmt.Fprintf(w, "\t\t%s\t%v\n", addr, lastSeen)
				}
			}
			return nil
		},
	}
}
