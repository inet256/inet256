package inet256cmd

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.inet256.org/inet256/src/inet256"
)

func NewKeygenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keygen",
		Short: "generates a private key and writes it to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			privKey := generateKey()
			data := inet256.MarshalPrivateKey(nil, privKey)
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
		id := inet256.NewID(privateKey.Public().(inet256.PublicKey))
		out := cmd.OutOrStdout()
		data, _ := id.MarshalText()
		data = append(data, '\n')
		_, err = out.Write(data)
		return err
	}
	return c
}

func generateKey() inet256.PrivateKey {
	_, priv, err := inet256.GenerateKey()
	if err != nil {
		panic(err)
	}
	return priv
}

func loadPrivateKeyFromFile(p string) (inet256.PrivateKey, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return inet256.ParsePrivateKey(data)
}
