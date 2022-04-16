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
	return NewRootCmd().Execute()
}

func NewRootCmd() *cobra.Command {
	newClient := func() (inet256srv.Service, error) {
		return inet256client.NewExtendedClient(defaultAPIAddr)
	}
	newNode := func(ctx context.Context, privateKey p2p.PrivateKey) (inet256.Node, error) {
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
	c.AddCommand(newStatusCmd(newClient))
	c.AddCommand(newNetworksCmd())
	c.AddCommand(newIslandCmd())
	c.AddCommand(newDaemonCmd())
	c.AddCommand(newCreateConfigCmd())
	c.AddCommand(newCentralDiscoveryCmd())

	c.AddCommand(NewPingCmd(newNode))
	c.AddCommand(NewNetCatCmd(newNode))
	c.AddCommand(NewEchoCmd(newNode))
	c.AddCommand(NewIP6PortalCmd(newNode))
	c.AddCommand(NewIP6AddrCmd())
	c.AddCommand(NewKeygenCmd())
	c.AddCommand(NewAddrCmd())

	return c
}

type NodeFactory = func(context.Context, inet256.PrivateKey) (inet256.Node, error)
