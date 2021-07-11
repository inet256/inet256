package inet256cmd

import (
	"context"

	"github.com/inet256/inet256/networks/floodnet"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(islandCmd)
}

var islandCmd = &cobra.Command{
	Use:   "island",
	Short: "runs the INET256 API without any peers",
	RunE: func(cmd *cobra.Command, args []string) error {
		privateKey := generateKey()
		d := inet256d.New(inet256d.Params{
			APIAddr: defaultAPIAddr,
			MainNodeParams: inet256srv.Params{
				Networks: []inet256.NetworkSpec{
					{Index: 0, Name: "flood", Factory: floodnet.Factory},
				},
				PrivateKey: privateKey,
				Peers:      inet256srv.NewPeerStore(),
			},
		})
		return d.Run(context.Background())
	},
}
