package inet256cmd

import (
	"net"

	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const defaultAPIAddr = "127.0.0.1:25600"

func init() {
	rootCmd.AddCommand(daemonCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Runs the inet256 daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		node, _, config, err := setupNode(cmd, args)
		if err != nil {
			return err
		}
		logrus.Println("LOCAL ID: ", node.LocalAddr())
		defer func() {
			if err := node.Close(); err != nil {
				logrus.Error(err)
			}
		}()
		l, err := net.Listen("tcp", config.GetAPIAddr())
		if err != nil {
			return err
		}
		defer l.Close()
		logrus.Println("API listening on: ", l.Addr())
		gs := grpc.NewServer()
		s := inet256grpc.NewServer(node)
		inet256grpc.RegisterINET256Server(gs, s)
		return gs.Serve(l)
	},
}
