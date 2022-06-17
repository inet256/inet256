package inet256cmd

import (
	"context"
	"fmt"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/spf13/cobra"
)

func NewPingCmd(newNode NodeFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "ping <target_addr>",
		Short: "ping an inet256 node",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			privKey := generateKey()
			node, err := newNode(ctx, privKey)
			if err != nil {
				return err
			}
			dst, err := inet256.ParseAddrBase64([]byte(args[0]))
			if err != nil {
				return err
			}
			fmt.Println("pinging", dst, "...")
			return node.Send(ctx, dst, []byte("ping"))
		},
	}
}
