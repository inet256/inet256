package inet256cmd

import (
	"context"

	"github.com/inet256/inet256/pkg/inet256ipv6"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(portalCmd)
}

var portalCmd = &cobra.Command{
	Use:   "ipv6-portal",
	Short: "runs an IPv6 portal",
	RunE: func(cmd *cobra.Command, args []string) error {
		privateKey := generateKey()
		n, err := newClient(privateKey)
		if err != nil {
			return err
		}
		ctx := context.Background()
		return inet256ipv6.RunPortal(ctx, inet256ipv6.PortalParams{
			Network: n,
			Logger:  logrus.New(),
		})
	},
}
