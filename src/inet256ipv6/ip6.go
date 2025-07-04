package inet256ipv6

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.inet256.org/inet256/src/inet256"
)

func NewIP6PortalCmd(newNode func(context.Context, inet256.PrivateKey) (inet256.Node, error)) *cobra.Command {
	c := &cobra.Command{
		Use:   "ip6-portal",
		Short: "runs an IP6 portal",
	}
	privateKeyPath := c.Flags().String("private-key", "", "--private-key=path/to/key.pem")
	whitelistPath := c.Flags().String("whitelist", "", "--whitelist=path/to/whitelist.txt")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if *privateKeyPath == "" {
			return errors.New("must provide path to private key")
		}
		privateKey, err := loadPrivateKeyFromFile(*privateKeyPath)
		if err != nil {
			return err
		}
		ctx := context.Background()
		n, err := newNode(ctx, privateKey)
		if err != nil {
			return err
		}
		allowFunc := AllowAll
		if *whitelistPath != "" {
			data, err := os.ReadFile(*whitelistPath)
			if err != nil {
				return err
			}
			allowFunc, err = ParseWhitelist(data)
			if err != nil {
				return err
			}
		}
		return RunPortal(ctx, PortalParams{
			AllowFunc: allowFunc,
			Node:      n,
		})
	}
	return c
}

func NewIP6AddrCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "ip6-addr",
		Short: "writes the ipv6 address corresponding to a private or public key",
	}
	privateKeyPath := c.Flags().String("private-key", "", "--private-key=path/to/key.pem")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		var x inet256.Addr
		switch {
		case *privateKeyPath != "":
			privateKey, err := loadPrivateKeyFromFile(*privateKeyPath)
			if err != nil {
				return err
			}
			x = inet256.NewID(privateKey.Public().(inet256.PublicKey))
		case len(args) == 1:
			var err error
			x, err = inet256.ParseAddrBase64([]byte(args[0]))
			if err != nil {
				return err
			}
		default:
			return errors.Errorf("must specify a key")
		}
		y := IPv6FromINET256(x)
		fmt.Fprintf(out, "%v\n", y)
		return nil
	}
	return c
}

func loadPrivateKeyFromFile(p string) (inet256.PrivateKey, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return inet256.ParsePrivateKey(data)
}
