package inet256cmd

import (
	"context"
	"io/ioutil"

	"github.com/inet256/inet256/pkg/inet256ipv6"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(portalCmd)
	rootCmd.AddCommand(createPortalConfigCmd)

	portalCmd.Flags().StringVar(&configPath, "config", "", "--config=./my-config.yml")
}

var portalCmd = &cobra.Command{
	Use:   "ip6-portal",
	Short: "runs an IP6 portal",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		if configPath == "" {
			return errors.New("must provide config path")
		}
		configData, err := ioutil.ReadFile(configPath)
		if err != nil {
			return err
		}
		config, err := inet256ipv6.ParsePortalConfig(configData)
		if err != nil {
			return err
		}
		privateKey, err := config.GetPrivateKey()
		if err != nil {
			return err
		}
		n, err := newClient(privateKey)
		if err != nil {
			return err
		}
		ctx := context.Background()
		return inet256ipv6.RunPortal(ctx, inet256ipv6.PortalParams{
			AllowFunc: config.GetAllowFunc(),
			Network:   n,
			Logger:    logrus.New(),
		})
	},
}

var createPortalConfigCmd = &cobra.Command{
	Use:   "ip6-create-config",
	Short: "writes a default config for an IPv6 portal to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		config := inet256ipv6.DefaultPortalConfig()
		data, err := yaml.Marshal(config)
		if err != nil {
			panic(err)
		}
		out.Write(data)
		return nil
	},
}
