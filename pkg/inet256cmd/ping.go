package inet256cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/client/go_client/inet256client"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pingCmd)
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "ping an inet256 node",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		privKey := generateKey()
		node, err := newNode(ctx, privKey)
		if err != nil {
			return err
		}
		if len(args) < 1 {
			return errors.New("must provide target addr")
		}

		dst := inet256.Addr{}
		if n, err := base64.RawURLEncoding.Decode(dst[:], []byte(args[0])); err != nil {
			return err
		} else if n != len(dst) {
			return errors.New("arg not long enough to be address")
		}

		fmt.Println("pinging", dst)
		return node.Tell(ctx, dst, []byte("ping"))
	},
}

func newClient() (inet256srv.Service, error) {
	return inet256client.NewExtendedClient(defaultAPIAddr)
}

func newNode(ctx context.Context, privateKey p2p.PrivateKey) (inet256.Network, error) {
	c, err := newClient()
	if err != nil {
		return nil, err
	}
	return c.CreateNode(ctx, privateKey)
}
