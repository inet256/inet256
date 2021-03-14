package inet256cmd

import (
	"context"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(echoCmd)
}

var echoCmd = &cobra.Command{
	Use:   "echo",
	Short: "echo starts a server which echos all messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		pk := generateKey()
		n, err := newClient(pk)
		if err != nil {
			return err
		}
		defer n.Close()
		logrus.Info(n.LocalAddr())
		ctx := context.Background()
		return n.Recv(func(src, dst inet256.Addr, data []byte) {
			if err := n.Tell(ctx, src, data); err != nil {
				logrus.Error(err)
				return
			}
			logrus.Infof("echoed %d bytes from %v", len(data), src)
		})
	},
}
