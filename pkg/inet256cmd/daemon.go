package inet256cmd

import (
	"context"

	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const defaultAPIAddr = inet256d.DefaultAPIEndpoint

func init() {
	daemonCmd.Flags().StringVar(&configPath, "config", "", "--config=./path/to/config/yaml")
	rootCmd.AddCommand(daemonCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Runs the inet256 daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}
		if configPath == "" {
			return errors.New("must provide config path")
		}
		config, err := inet256d.LoadConfig(configPath)
		if err != nil {
			return err
		}
		log.Infof("using config from path: %v", configPath)
		params, err := inet256d.MakeParams(configPath, *config)
		if err != nil {
			return err
		}
		d := inet256d.New(*params)
		return d.Run(context.Background())
	},
}

func setupDaemon(cmd *cobra.Command, args []string) (*inet256d.Daemon, error) {
	if err := cmd.ParseFlags(args); err != nil {
		return nil, err
	}
	if configPath == "" {
		return nil, errors.New("must provide config path")
	}
	log.Infof("using config path: %v", configPath)
	config, err := inet256d.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	params, err := inet256d.MakeParams(configPath, *config)
	if err != nil {
		return nil, err
	}
	return inet256d.New(*params), nil
}
