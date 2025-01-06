package inet256cmd

import (
	"context"

	"github.com/spf13/cobra"
	"go.brendoncarroll.net/stdctx/logctx"
	"go.uber.org/zap"

	"go.inet256.org/inet256/client/go/inet256client"
	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/inet256d"
	"go.inet256.org/inet256/pkg/inet256ipv6"
)

const defaultAPIAddr = inet256d.DefaultAPIEndpoint

var ctx = func() context.Context {
	ctx := context.Background()
	l, _ := zap.NewProduction()
	ctx = logctx.NewContext(ctx, l)
	return ctx
}()

func Execute() error {
	return NewRootCmd().Execute()
}

func NewRootCmd() *cobra.Command {
	newClient := func() (inet256.Service, error) {
		return inet256client.NewEnvClient()
	}
	newAdminClient := func() (inet256d.AdminClient, error) {
		return inet256d.NewAdminClient(inet256client.GetAPIEndpointFromEnv())
	}
	newNode := func(ctx context.Context, privateKey inet256.PrivateKey) (inet256.Node, error) {
		c, err := newClient()
		if err != nil {
			return nil, err
		}
		return c.Open(ctx, privateKey)
	}
	c := &cobra.Command{
		Use:   "inet256",
		Short: "inet256: A secure network with a 256 bit address space",
	}
	c.AddCommand(newStatusCmd(newAdminClient))
	c.AddCommand(newNetworksCmd())
	c.AddCommand(newIslandCmd())
	c.AddCommand(newDaemonCmd())
	c.AddCommand(newCreateConfigCmd())
	c.AddCommand(newCentralDiscoveryCmd())

	c.AddCommand(NewPingCmd(newNode))
	c.AddCommand(NewNetCatCmd(newNode))
	c.AddCommand(NewEchoCmd(newNode))
	c.AddCommand(inet256ipv6.NewIP6PortalCmd(newNode))
	c.AddCommand(inet256ipv6.NewIP6AddrCmd())
	c.AddCommand(NewKeygenCmd())
	c.AddCommand(NewAddrCmd())
	c.AddCommand(NewDialCmd(newNode))
	c.AddCommand(NewListenCmd(newNode))

	return c
}

type NodeFactory = func(context.Context, inet256.PrivateKey) (inet256.Node, error)
