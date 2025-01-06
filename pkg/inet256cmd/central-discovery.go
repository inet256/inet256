package inet256cmd

import (
	"net"

	"github.com/spf13/cobra"
	"go.brendoncarroll.net/stdctx/logctx"
	"google.golang.org/grpc"

	"go.inet256.org/inet256/pkg/discovery"
	"go.inet256.org/inet256/pkg/discovery/centraldisco"
)

func newCentralDiscoveryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "central-discovery",
		Short: "runs a central discovery server",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			laddr := args[0]
			s := centraldisco.NewServer(func([]byte) (discovery.TransportAddr, error) {
				return discovery.TransportAddr{}, nil
			})
			gs := grpc.NewServer()
			centraldisco.RegisterServer(gs, s)
			l, err := net.Listen("tcp", laddr)
			if err != nil {
				return err
			}
			defer l.Close()
			logctx.Infof(ctx, "serving on %s...", l.Addr().String())
			return gs.Serve(l)
		},
	}
}
