package inet256cmd

import (
	"context"
	"encoding/base64"
	"errors"
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
			dst := inet256.Addr{}
			if n, err := base64.RawURLEncoding.Decode(dst[:], []byte(args[0])); err != nil {
				return err
			} else if n != len(dst) {
				return errors.New("arg not long enough to be address")
			}
			fmt.Println("pinging", dst, "...")
			return node.Send(ctx, dst, []byte("ping"))
		},
	}
}
