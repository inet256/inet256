package inet256cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(keygenCmd)
}

var keygenCmd = &cobra.Command{
	Use: "keygen",
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

func generateKey() (p2p.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}
