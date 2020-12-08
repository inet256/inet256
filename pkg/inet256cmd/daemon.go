package inet256cmd

import (
	"context"
	"net"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const defaultAPIAddr = "127.0.0.1:25600"

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
		config, err := LoadConfig(configPath)
		if err != nil {
			return err
		}
		log.Infof("using config from path: %v", configPath)
		d := newDaemon(configPath, *config)
		return d.run(context.Background())
	},
}

type daemon struct {
	configPath string
	config     Config
	log        *logrus.Logger
}

func newDaemon(configPath string, config Config) *daemon {
	d := &daemon{
		configPath: configPath,
		config:     config,
		log:        logrus.New(),
	}
	return d
}

func (d *daemon) run(ctx context.Context) error {
	nodeParams, err := MakeNodeParams(d.configPath, d.config)
	if err != nil {
		return err
	}
	localID := p2p.NewPeerID(nodeParams.PrivateKey.Public())

	// discovery
	peerStores := []inet256.PeerStore{nodeParams.Peers}
	dscSrvs := []p2p.DiscoveryService{}
	for _, spec := range d.config.Discovery {
		d, err := makeDiscoveryService(spec)
		if err != nil {
			return err
		}
		dscSrvs = append(dscSrvs, d)
		peerStores = append(peerStores, inet256.NewPeerStore())
	}
	nodeParams.Peers = inet256.ChainPeerStore(peerStores)

	// node
	node := inet256.NewNode(*nodeParams)
	defer func() {
		if err := node.Close(); err != nil {
			d.log.Error(err)
		}
	}()
	d.log.Println("LOCAL ID: ", node.LocalAddr())

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return d.runGRPCServer(ctx, d.config.APIAddr, node)
	})
	eg.Go(func() error {
		return d.runDiscoveryServices(ctx, localID, dscSrvs, node.TransportAddrs, peerStores[1:])
	})
	return eg.Wait()
}

func (d *daemon) runGRPCServer(ctx context.Context, endpoint string, node *inet256.Node) error {
	l, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	defer l.Close()
	logrus.Println("API listening on: ", l.Addr())
	gs := grpc.NewServer()
	s := inet256grpc.NewServer(node)
	inet256grpc.RegisterINET256Server(gs, s)
	go func() {
		<-ctx.Done()
		gs.Stop()
	}()
	return gs.Serve(l)
}

func (d *daemon) runDiscoveryServices(ctx context.Context, localID p2p.PeerID, ds []p2p.DiscoveryService, localAddrs func() []string, ps []inet256.PeerStore) error {
	eg := errgroup.Group{}
	for i := range ds {
		disc := ds[i]
		p := ps[i]
		eg.Go(func() error {
			return d.runDiscoveryService(ctx, localID, disc, localAddrs, p.(inet256.MutablePeerStore))
		})
	}
	return eg.Wait()
}

func (d *daemon) runDiscoveryService(ctx context.Context, localID p2p.PeerID, ds p2p.DiscoveryService, localAddrs func() []string, ps inet256.MutablePeerStore) error {
	eg := errgroup.Group{}
	// announce loop
	eg.Go(func() error {
		ttl := 60 * time.Second
		ticker := time.NewTicker(60 * time.Second / 2)
		defer ticker.Stop()
		for {
			addrs := localAddrs()
			if err := ds.Announce(ctx, localID, addrs, ttl); err != nil {
				logrus.Error("announce error:", err)
			}
			// TODO: announce, and find, add each to the store
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	})
	// Find loop
	eg.Go(func() error {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			for _, id := range ps.ListPeers() {
				addrs, err := ds.Find(ctx, localID)
				if err != nil {
					logrus.Error("find errored: ", err)
					continue
				}
				ps.PutAddrs(id, addrs)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	})
	return eg.Wait()
}
