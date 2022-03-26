package inet256cmd

import (
	"net"

	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/discovery/centraldisco"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func init() {
	rootCmd.AddCommand(centralDiscoveryCmd)
}

var centralDiscoveryCmd = &cobra.Command{
	Use:   "central-discovery",
	Short: "runs a central discovery server",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		laddr := args[0]
		logger := logrus.StandardLogger()
		s := centraldisco.NewServer(logger, func([]byte) (discovery.TransportAddr, error) {
			return discovery.TransportAddr{}, nil
		})
		gs := grpc.NewServer()
		centraldisco.RegisterServer(gs, s)
		l, err := net.Listen("tcp", laddr)
		if err != nil {
			return err
		}
		defer l.Close()
		logger.Infof("serving on %s...", l.Addr().String())
		return gs.Serve(l)
	},
}
