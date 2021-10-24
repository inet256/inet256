package inet256cmd

import (
	"crypto/ed25519"
	"crypto/rand"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(keygenCmd)
	rootCmd.AddCommand(addrCmd)

	addrCmd.Flags().StringVar(&privateKeyPath, "private-key", "", "--private-key path/to/private/key.pem")
}

var keygenCmd = &cobra.Command{
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

var addrCmd = &cobra.Command{
	Use:   "addr",
	Short: "derives an address from a private key",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		if privateKeyPath == "" {
			return errors.Errorf("must provide path to private key")
		}
		privateKey, err := getPrivateKeyFromFile(privateKeyPath)
		if err != nil {
			return err
		}
		id := inet256.NewAddr(privateKey.Public())
		out := cmd.OutOrStdout()
		data, _ := id.MarshalText()
		data = append(data, '\n')
		_, err = out.Write(data)
		return err
	},
}

func generateKey() p2p.PrivateKey {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return priv
}
