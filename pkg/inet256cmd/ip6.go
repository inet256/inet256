package inet256cmd

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256ipv6"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var privateKeyPath string
var whitelistPath string

func init() {
	rootCmd.AddCommand(portalCmd)
	rootCmd.AddCommand(ip6AddrCmd)

	portalCmd.Flags().StringVar(&privateKeyPath, "private-key", "", "--private-key=path/to/key.pem")
	portalCmd.Flags().StringVar(&whitelistPath, "whitelist", "", "--whitelist=path/to/whitelist.txt")

	ip6AddrCmd.Flags().StringVar(&privateKeyPath, "private-key", "", "--private-key=path/to/key.pem")
}

var portalCmd = &cobra.Command{
	Use:   "ip6-portal",
	Short: "runs an IP6 portal",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		if privateKeyPath == "" {
			return errors.New("must provide path to private key")
		}
		privateKey, err := getPrivateKeyFromFile(privateKeyPath)
		if err != nil {
			return err
		}
		ctx := context.Background()
		n, err := newNode(ctx, privateKey)
		if err != nil {
			return err
		}
		allowFunc := inet256ipv6.AllowAll
		if whitelistPath != "" {
			data, err := ioutil.ReadFile(whitelistPath)
			if err != nil {
				return err
			}
			allowFunc, err = inet256ipv6.ParseWhitelist(data)
			if err != nil {
				return err
			}
		}
		return inet256ipv6.RunPortal(ctx, inet256ipv6.PortalParams{
			AllowFunc: allowFunc,
			Node:      n,
			Logger:    logrus.New(),
		})
	},
}

var ip6AddrCmd = &cobra.Command{
	Use:   "ip6-addr",
	Short: "writes the ipv6 address corresponding to a private or public key",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		var x inet256.Addr
		switch {
		case privateKeyPath != "":
			privateKey, err := getPrivateKeyFromFile(privateKeyPath)
			if err != nil {
				return err
			}
			x = inet256.NewAddr(privateKey.Public())
		case len(args) == 1:
			var err error
			x, err = inet256.ParseAddrB64([]byte(args[0]))
			if err != nil {
				return err
			}
		default:
			return errors.Errorf("must specify a key")
		}
		y := inet256ipv6.INet256ToIPv6(x)
		fmt.Fprintf(out, "%v\n", y)
		return nil
	},
}

func getPrivateKeyFromFile(p string) (inet256.PrivateKey, error) {
	data, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}
	return serde.ParsePrivateKeyPEM(data)
}
