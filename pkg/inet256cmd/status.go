package inet256cmd

import (
	"fmt"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "prints status of the main node",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		var localAddr inet256.Addr
		var transportAddrs []p2p.Addr
		var peerStatuses []inet256srv.PeerStatus
		eg := errgroup.Group{}
		eg.Go(func() error {
			localAddr = c.MainAddr()
			return nil
		})
		eg.Go(func() error {
			transportAddrs = c.TransportAddrs()
			return nil
		})
		eg.Go(func() error {
			peerStatuses = c.PeerStatus()
			return nil
		})
		if err := eg.Wait(); err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "LOCAL ADDR: %v\n", localAddr)
		fmt.Fprintf(w, "TRANSPORTS:\n")
		for _, addr := range transportAddrs {
			fmt.Fprintf(w, "\t%s\n", addr)
		}
		fmt.Fprintf(w, "PEERS:\n")
		for _, status := range peerStatuses {
			fmt.Fprintf(w, "\t%s\n", status.Addr)
			for addr, lastSeen := range status.LastSeen {
				fmt.Fprintf(w, "\t\t%s\t%v\n", addr, lastSeen)
			}
		}
		return nil
	},
}
