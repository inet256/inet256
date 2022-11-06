package inet256d

import (
	"context"
	"net"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/inet256/inet256/pkg/autopeering"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/peers"
)

type TransportAddr = mesh256.TransportAddr
type PeerStore = peers.Store[TransportAddr]

type Params struct {
	MainNodeParams      mesh256.Params
	DiscoveryServices   []discovery.Service
	AutoPeeringServices []autopeering.Service
	APIAddr             string
	TransportAddrParser p2p.AddrParser[TransportAddr]
}

type Daemon struct {
	params Params
	log    *logrus.Logger

	setupDone chan struct{}
	s         *mesh256.Server
}

func New(p Params) *Daemon {
	return &Daemon{
		params:    p,
		log:       logrus.New(),
		setupDone: make(chan struct{}),
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	promReg := prometheus.NewRegistry()
	promReg.Register(prometheus.NewGoCollector())

	nodeParams := d.params.MainNodeParams
	localID := inet256.NewAddr(nodeParams.PrivateKey.Public())

	// discovery
	dscSrvs := d.params.DiscoveryServices
	dscPeerStores := make([]PeerStore, len(dscSrvs))
	for i := range dscSrvs {
		// initialize and copy peers, since discovery services don't add peers.
		dscPeerStores[i] = mesh256.NewPeerStore()
		copyPeers(dscPeerStores[i], d.params.MainNodeParams.Peers)
	}

	// auto-peering
	apSrvs := d.params.AutoPeeringServices
	apPeerStores := make([]PeerStore, len(apSrvs))
	for i := range apSrvs {
		apPeerStores[i] = mesh256.NewPeerStore()
	}

	peerStores := []PeerStore{d.params.MainNodeParams.Peers}
	peerStores = append(peerStores, dscPeerStores...)
	peerStores = append(peerStores, apPeerStores...)

	// server
	nodeParams.Peers = peers.ChainStore[TransportAddr](peerStores)
	s := mesh256.NewServer(nodeParams)
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
		return d.runHTTPServer(ctx, d.params.APIAddr, s, promReg)
	})
	eg.Go(func() error {
		d.runDiscoveryServices(ctx, d.params.MainNodeParams.PrivateKey, d.params.DiscoveryServices, adaptTransportAddrs(s.TransportAddrs), dscPeerStores, d.params.TransportAddrParser)
		return nil
	})
	eg.Go(func() error {
		d.runAutoPeeringServices(ctx, localID, d.params.AutoPeeringServices, apPeerStores, adaptTransportAddrs(s.TransportAddrs))
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

func (d *Daemon) runGRPCServer(ctx context.Context, endpoint string, s *mesh256.Server) error {
	l, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	defer l.Close()
	d.log.Println("API listening on: ", l.Addr())
	gs := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    1 * time.Second,
		Timeout: 5 * time.Second,
	}))
	srv := inet256grpc.NewServer(s)
	inet256grpc.RegisterINET256Server(gs, srv)
	go func() {
		<-ctx.Done()
		gs.Stop()
	}()
	return gs.Serve(l)
}

func copyPeers(dst, src peers.Store[mesh256.TransportAddr]) {
	for _, id := range src.ListPeers() {
		dst.Add(id)
	}
}
