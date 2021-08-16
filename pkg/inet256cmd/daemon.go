package inet256cmd

import (
	"context"

	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const defaultAPIAddr = inet256d.DefaultAPIAddr

func init() {
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
