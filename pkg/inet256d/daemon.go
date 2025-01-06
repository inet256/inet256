package inet256d

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.brendoncarroll.net/p2p"
	"go.brendoncarroll.net/stdctx/logctx"
	"golang.org/x/sync/errgroup"

	"go.inet256.org/inet256/pkg/discovery"
	"go.inet256.org/inet256/pkg/mesh256"
	"go.inet256.org/inet256/pkg/peers"
)

type TransportAddr = mesh256.TransportAddr
type PeerStore = peers.Store[TransportAddr]

type Params struct {
	MainNodeParams      mesh256.Params
	Discovery           []discovery.Service
	APIAddr             string
	TransportAddrParser p2p.AddrParser[TransportAddr]
}

type Daemon struct {
	params Params

	setupDone chan struct{}
	s         *mesh256.Server
}

func New(p Params) *Daemon {
	return &Daemon{
		params:    p,
		setupDone: make(chan struct{}),
	}
}

// Run runs the daemon.
// Logs are written to the slog.Logger in the context.
func (d *Daemon) Run(ctx context.Context) error {
	promReg := prometheus.NewRegistry()
	promReg.Register(collectors.NewGoCollector())

	nodeParams := d.params.MainNodeParams
	nodeParams.Background = ctx

	// discovery
	dscSrvs := d.params.Discovery
	dscPeerStores := make([]PeerStore, len(dscSrvs))
	for i := range dscSrvs {
		// initialize and copy peers, since discovery services don't add peers.
		dscPeerStores[i] = mesh256.NewPeerStore()
		copyPeers(dscPeerStores[i], d.params.MainNodeParams.Peers)
	}

	peerStores := []PeerStore{d.params.MainNodeParams.Peers}
	peerStores = append(peerStores, dscPeerStores...)

	// server
	nodeParams.Peers = peers.ChainStore[TransportAddr](peerStores)
	s := mesh256.NewServer(nodeParams)
	defer func() {
		if err := s.Close(); err != nil {
			logctx.Errorln(ctx, "closing service", err)
		}
	}()
	d.s = s
	close(d.setupDone)
	logctx.Infoln(ctx, "LOCAL ID: ", s.LocalAddr())

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return d.runHTTPServer(ctx, d.params.APIAddr, s, promReg)
	})
	eg.Go(func() error {
		d.runDiscovery(ctx, d.params.MainNodeParams.PrivateKey, d.params.Discovery, adaptTransportAddrs(s.TransportAddrs), dscPeerStores, d.params.TransportAddrParser)
		return nil
	})
	return eg.Wait()
}

func (d *Daemon) DoWithServer(ctx context.Context, cb func(s *mesh256.Server) error) error {
	select {
	case <-d.setupDone:
		return cb(d.s)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func copyPeers(dst, src peers.Store[mesh256.TransportAddr]) {
	for _, id := range src.List() {
		dst.Add(id)
	}
}
