package inet256cmd

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pingCmd)
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "ping an inet256 node",
	RunE: func(cmd *cobra.Command, args []string) error {
		node, _, err := setupNode(cmd, args)
		if err != nil {
			return err
		}

		dst := inet256.Addr{}
		base64.RawURLEncoding.Decode(dst[:], []byte(args[0]))

		ctx := context.Background()
		fmt.Println("pinging", dst)
		return node.Tell(ctx, dst, []byte("ping"))
	},
}
