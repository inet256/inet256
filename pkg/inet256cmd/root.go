package inet256cmd

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/client/go_client/inet256client"
	"github.com/spf13/cobra"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/inet256ipv6"
)

const defaultAPIAddr = "http://127.0.0.1:2560"

func Execute() error {
	return NewRootCmd().Execute()
}

func NewRootCmd() *cobra.Command {
	newClient := func() (inet256.Service, error) {
		return inet256client.NewEnvClient()
	}
	newAdminClient := func() (inet256d.AdminClient, error) {
		return inet256d.NewAdminClient(defaultAPIAddr + "/admin")
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

	return c
}

type NodeFactory = func(context.Context, inet256.PrivateKey) (inet256.Node, error)
