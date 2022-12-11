package inet256cmd

import (
	"context"

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
		return d.Run(context.Background())
	}
	return c
}

func newCreateConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create-config",
		Short: "creates a new default config and writes it to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := inet256d.DefaultConfig()
			data, err := yaml.Marshal(c)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			out.Write(data)
			return nil
		},
	}
}
