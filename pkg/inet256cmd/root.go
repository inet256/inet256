package inet256cmd

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/client/go_client/inet256client"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/spf13/cobra"
)

func Execute() error {
	return rootCmd.Execute()
}

var (
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "inet256",
	Short: "inet256: A secure network with a 256 bit address space",
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
