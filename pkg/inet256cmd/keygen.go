package inet256cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"io/ioutil"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(keygenCmd)
	rootCmd.AddCommand(deriveAddrCmd)
}

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "generates a private key and writes it to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		privKey, err := generateKey()
		if err != nil {
			return err
		}
		data, err := inet256.MarshalPrivateKeyPEM(privKey)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(data)
		return nil
	},
}

var deriveAddrCmd = &cobra.Command{
	Use:   "derive-addr",
	Short: "derives an address from a private key, read from stdin",
	RunE: func(cmd *cobra.Command, args []string) error {
		in := cmd.InOrStdin()
		data, err := ioutil.ReadAll(in)
		if err != nil {
			return err
		}
		privateKey, err := inet256.ParsePrivateKeyPEM(data)
		if err != nil {
			return err
		}
		id := inet256.NewAddr(privateKey.Public())
		out := cmd.OutOrStdout()
		data, _ = id.MarshalText()
		data = append(data, '\n')
		out.Write(data)
		return nil
	},
}

func generateKey() (p2p.PrivateKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	return priv, err
}
