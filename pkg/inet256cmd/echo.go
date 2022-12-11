package inet256cmd

import (
	"context"

	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/spf13/cobra"

	"github.com/inet256/inet256/pkg/inet256"
)

func NewEchoCmd(newNode NodeFactory) *cobra.Command {
	return &cobra.Command{
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
			logctx.Infoln(ctx, node.LocalAddr())
			var msg inet256.Message
			for {
				if err := inet256.Receive(ctx, node, &msg); err != nil {
					return err
				}
				if err := node.Send(ctx, msg.Src, msg.Payload); err != nil {
					logctx.Errorln(ctx, err)
					continue
				}
				logctx.Infof(ctx, "echoed %d bytes from %v", len(msg.Payload), msg.Src)
			}
		},
	}
}
