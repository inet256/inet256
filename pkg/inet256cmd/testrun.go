package inet256cmd

import (
	"context"
	"errors"
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(testRunCmd)
}

var testRunCmd = &cobra.Command{
	Use:   "testrun",
	Short: "run an inet256 node with logging",
	RunE: func(cmd *cobra.Command, args []string) error {
		node, params, _, err := setupNode(cmd, args)
		if err != nil {
			return err
		}

		cmd.Printf("ADDR: %v\n", node.LocalAddr())
		cmd.Printf("LISTENERS: %v\n", params.Swarms)
		node.OnRecv(func(src, dst inet256.Addr, data []byte) {
			cmd.Printf("RECV: src=%v dst=%v data=%v\n", src, dst, string(data))
		})

		ctx := context.Background()
		data := []byte("ping")
		for {
			for _, addr := range params.Peers.ListPeers() {
				if err := node.Tell(ctx, addr, data); err != nil {
					log.Error(err)
					continue
				} else {
					// fmt.Printf("SEND: src=%v dst=%v data=%v\n", node.LocalAddr(), addr, string(data))
				}
			}
			time.Sleep(time.Second)
		}
		return nil
	},
}

func setupNode(cmd *cobra.Command, args []string) (*inet256.Node, *inet256.Params, *Config, error) {
	if err := cmd.ParseFlags(args); err != nil {
		return nil, nil, nil, err
	}
	if configPath == "" {
		return nil, nil, nil, errors.New("must provide config path")
	}
	log.Infof("using config path: %v", configPath)
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, nil, nil, err
	}
	params, err := BuildParams(configPath, config)
	if err != nil {
		return nil, nil, nil, err
	}
	node := inet256.NewNode(*params)
	return node, params, config, nil
}
