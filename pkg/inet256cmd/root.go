package inet256cmd

import (
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func Execute() error {
	return rootCmd.Execute()
}

var (
	configPath string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "--config=./path/to/config/yaml")
}

var rootCmd = &cobra.Command{
	Use:   "inet256",
	Short: "inet256: A secure network with a 256 bit address space",
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
