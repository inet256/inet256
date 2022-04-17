package inet256cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"io/ioutil"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewKeygenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keygen",
		Short: "generates a private key and writes it to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			privKey := generateKey()
			data, err := serde.MarshalPrivateKeyPEM(privKey)
			if err != nil {
				return err
			}
			cmd.OutOrStdout().Write(data)
			return nil
		},
	}
}

func NewAddrCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "addr",
		Short: "derives an address from a private key",
	}
	privateKeyPath := c.Flags().String("private-key", "", "--private-key path/to/private/key.pem")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		if *privateKeyPath == "" {
			return errors.Errorf("must provide path to private key")
		}
		privateKey, err := loadPrivateKeyFromFile(*privateKeyPath)
		if err != nil {
			return err
		}
		id := inet256.NewAddr(privateKey.Public())
		out := cmd.OutOrStdout()
		data, _ := id.MarshalText()
		data = append(data, '\n')
		_, err = out.Write(data)
		return err
	}
	return c
}

func generateKey() p2p.PrivateKey {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return priv
}

func loadPrivateKeyFromFile(p string) (inet256.PrivateKey, error) {
	data, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return serde.ParsePrivateKeyPEM(data)
}
