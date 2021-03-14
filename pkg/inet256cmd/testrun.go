package inet256cmd

import (
	"context"
	"errors"
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256d"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	rootCmd.AddCommand(testRunCmd)
}

var testRunCmd = &cobra.Command{
	Use:   "testrun",
	Short: "run an inet256 node with logging",
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := setupDaemon(cmd, args)
		if err != nil {
			return err
		}
		ctx := context.Background()
		eg := errgroup.Group{}
		eg.Go(func() error {
			return d.Run(ctx)
		})
		eg.Go(func() error {
			return d.DoWithServer(ctx, func(s *inet256.Server) error {
				node := s.MainNode()
				cmd.Printf("ADDR: %v\n", s.LocalAddr())
				cmd.Printf("LISTENERS: %v\n", s.TransportAddrs())
				go node.Recv(func(src, dst inet256.Addr, data []byte) {
					cmd.Printf("RECV: src=%v dst=%v data=%v\n", src, dst, string(data))
				})

				data := []byte("ping")
				period := time.Second
				ticker := time.NewTicker(period)
				defer ticker.Stop()
				for range ticker.C {
					func() {
						ctx, cf := context.WithTimeout(ctx, period)
						cf()
						for _, addr := range node.ListOneHop() {
							if err := node.Tell(ctx, addr, data); err != nil {
								log.Error(err)
								continue
							} else {
								// fmt.Printf("SEND: src=%v dst=%v data=%v\n", node.LocalAddr(), addr, string(data))
							}
						}
					}()
				}
				return nil
			})
		})
		return eg.Wait()
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
