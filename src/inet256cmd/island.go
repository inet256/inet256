package inet256cmd

import (
	"github.com/spf13/cobra"
	"go.inet256.org/inet256/networks/beaconnet"
	"go.inet256.org/inet256/src/inet256d"
	"go.inet256.org/inet256/src/mesh256"
)

func newIslandCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "island",
		Short: "runs the INET256 API without any peers",
		RunE: func(cmd *cobra.Command, args []string) error {
			privateKey := generateKey()
			d := inet256d.New(inet256d.Params{
				APIAddr: defaultAPIAddr,
				MainNodeParams: mesh256.Params{
					NewNetwork: beaconnet.Factory,
					PrivateKey: privateKey,
					Peers:      mesh256.NewPeerStore(),
				},
			})
			return d.Run(ctx)
		},
	}
}
