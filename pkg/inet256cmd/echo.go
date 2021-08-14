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
		ctx := context.Background()
		pk := generateKey()
		node, err := newNode(ctx, pk)
		if err != nil {
			return err
		}
		defer node.Close()
		logrus.Info(node.LocalAddr())
		buf := make([]byte, inet256.TransportMTU)
		for {
			var src, dst inet256.Addr
			n, err := node.Receive(ctx, &src, &dst, buf)
			if err != nil {
				return err
			}
			data := buf[:n]
			if err := node.Tell(ctx, src, data); err != nil {
				logrus.Error(err)
				continue
			}
			logrus.Infof("echoed %d bytes from %v", len(data), src)
		}
	},
}
