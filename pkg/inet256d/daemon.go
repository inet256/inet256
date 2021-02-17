package inet256d

import (
	"context"
	"net"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/d/celltracker"
	"github.com/inet256/inet256/pkg/autopeering"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Params struct {
	MainNodeParams      inet256.Params
	DiscoveryServices   []p2p.DiscoveryService
	AutoPeeringServices []autopeering.Service
	APIAddr             string
}

type Daemon struct {
	params Params
	log    *logrus.Logger

	setupDone chan struct{}
	s         *inet256.Server
}

func New(p Params) *Daemon {
	return &Daemon{
		params:    p,
		log:       logrus.New(),
		setupDone: make(chan struct{}),
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	nodeParams := d.params.MainNodeParams
	localID := p2p.NewPeerID(nodeParams.PrivateKey.Public())
	dscSrvs := d.params.DiscoveryServices
	// discovery
	peerStores := []inet256.PeerStore{nodeParams.Peers}
	for range dscSrvs {
		peerStores = append(peerStores, inet256.NewPeerStore())
	}

	// server
	s := inet256.NewServer(d.params.MainNodeParams)
	defer func() {
		if err := s.Close(); err != nil {
			d.log.Error(err)
		}
	}()
	d.s = s
	close(d.setupDone)
	d.log.Println("LOCAL ID: ", s.LocalAddr())

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return d.runGRPCServer(ctx, d.params.APIAddr, s)
	})
	eg.Go(func() error {
		return d.runDiscoveryServices(ctx, localID, d.params.DiscoveryServices, s.TransportAddrs, peerStores[1:])
	})
	return eg.Wait()
}

func (d *Daemon) DoWithServer(ctx context.Context, cb func(s *inet256.Server) error) error {
	select {
	case <-d.setupDone:
		return cb(d.s)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Daemon) runGRPCServer(ctx context.Context, endpoint string, s *inet256.Server) error {
	l, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	defer l.Close()
	logrus.Println("API listening on: ", l.Addr())
	gs := grpc.NewServer()
	srv := inet256grpc.NewServer(s)
	inet256grpc.RegisterINET256Server(gs, srv)
	go func() {
		<-ctx.Done()
		gs.Stop()
	}()
	return gs.Serve(l)
}

func (d *Daemon) runDiscoveryServices(ctx context.Context, localID p2p.PeerID, ds []p2p.DiscoveryService, localAddrs func() []string, ps []inet256.PeerStore) error {
	eg := errgroup.Group{}
	for i := range ds {
		disc := ds[i]
		p := ps[i]
		eg.Go(func() error {
			return d.runDiscoveryService(ctx, localID, disc, localAddrs, p)
		})
	}
	return eg.Wait()
}

func (d *Daemon) runDiscoveryService(ctx context.Context, localID p2p.PeerID, ds p2p.DiscoveryService, localAddrs func() []string, ps inet256.PeerStore) error {
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
				ps.SetAddrs(id, addrs)
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

func makeDiscoveryService(spec DiscoverySpec) (p2p.DiscoveryService, error) {
	switch {
	case spec.CellTracker != nil:
		return celltracker.NewClient(*spec.CellTracker)
	default:
		return nil, errors.Errorf("empty discovery spec")
	}
}

func setupDiscovery(spec DiscoverySpec) (p2p.DiscoveryService, error) {
	switch {
	case spec.CellTracker != nil:
		return celltracker.NewClient(*spec.CellTracker)
	default:
		return nil, errors.Errorf("empty spec")
	}
}
