package inet256cmd

import (
	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/inet256/inet256/pkg/inet256d"
)

func newDaemonCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "daemon",
		Short: "Runs the inet256 daemon",
	}
	configPath := c.Flags().String("config", "", "--config=./path/to/config/yaml")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		if *configPath == "" {
			return errors.New("must provide config path")
		}
		config, err := inet256d.LoadConfig(*configPath)
		if err != nil {
			return err
		}
		logctx.Infof(ctx, "using config from path: %v", *configPath)
		params, err := inet256d.MakeParams(*configPath, *config)
		if err != nil {
			return err
		}
		d := inet256d.New(*params)
		return d.Run(ctx)
	}
	return c
}

func newCreateConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "create-config",
		Short: "creates a new default config and writes it to stdout",
	}
	api_endpoint := c.Flags().String("api_endpoint", "", "--api_endpoint=tcp://127.0.0.1:2560")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		c := inet256d.DefaultConfig()
		if *api_endpoint != "" {
			c.APIEndpoint = *api_endpoint
		}
		data, err := yaml.Marshal(c)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		out.Write(data)
		return nil
	}
	return c
}
