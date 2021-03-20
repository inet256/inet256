package inet256test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestServer(t *testing.T, nf inet256.NetworkFactory) {
	t.Run("Send", func(t *testing.T) {
		s := newTestServer(t, nf)
		testServerSend(t, s)
	})
	t.Run("Send", func(t *testing.T) {
		srvs := newTestServers(t, nf, 2)
		testMultipleServers(t, srvs...)
	})
}

func testServerSend(t *testing.T, s *inet256.Server) {
	ctx := context.Background()
	const N = 5
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := p2ptest.NewTestKey(t, i)
		n, err := s.CreateNode(ctx, pk)
		require.NoError(t, err)
		nodes[i] = n
	}
	t.Log("created", N, "nodes")
	chans := setupChans(castNodeSlice(nodes))
	randomPairs(len(nodes), func(i, j int) {
		testSendRecvOne(t, nodes[i], nodes[j].LocalAddr(), chans[j])
	})
}

func testMultipleServers(t *testing.T, srvs ...*inet256.Server) {
	ctx := context.Background()
	const N = 5
	nodes := make([]inet256.Node, len(srvs)*N)
	for i, s := range srvs {
		for j := 0; j < N; j++ {
			pk := p2ptest.NewTestKey(t, i*N+j)
			n, err := s.CreateNode(ctx, pk)
			require.NoError(t, err)
			nodes[i*N+j] = n
		}
	}
	t.Log("created", N, "nodes", "on each of", len(srvs), "servers")
	chans := setupChans(castNodeSlice(nodes))
	randomPairs(len(nodes), func(i, j int) {
		testSendRecvOne(t, nodes[i], nodes[j].LocalAddr(), chans[j])
	})
}

func newTestServer(t *testing.T, nf inet256.NetworkFactory) *inet256.Server {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	pk := p2ptest.NewTestKey(t, math.MaxInt32)
	ps := inet256.NewPeerStore()
	s := inet256.NewServer(inet256.Params{
		Networks: []inet256.NetworkSpec{
			{
				Factory: nf,
				Index:   0,
				Name:    "network-name",
			},
		},
		Peers:      ps,
		PrivateKey: pk,
	})
	err := s.MainNode().WaitReady(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})
	return s
}

func newTestServers(t *testing.T, nf inet256.NetworkFactory, n int) []*inet256.Server {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	r := memswarm.NewRealm()
	stores := make([]inet256.PeerStore, n)
	srvs := make([]*inet256.Server, n)
	for i := range srvs {
		pk := p2ptest.NewTestKey(t, math.MaxInt32+i)
		stores[i] = inet256.NewPeerStore()
		srvs[i] = inet256.NewServer(inet256.Params{
			Swarms: map[string]p2p.SecureSwarm{
				"external": r.NewSwarmWithKey(pk),
			},
			Networks: []inet256.NetworkSpec{
				{
					Factory: nf,
					Index:   0,
					Name:    "network-name",
				},
			},
			Peers:      stores[i],
			PrivateKey: pk,
		})
	}
	for i := range srvs {
		for j := range srvs {
			if i == j {
				continue
			}
			stores[i].Add(srvs[j].MainAddr())
			stores[i].SetAddrs(srvs[j].MainAddr(), srvs[j].TransportAddrs())
		}
	}
	t.Cleanup(func() {
		for _, s := range srvs {
			require.NoError(t, s.Close())
		}
	})
	eg := errgroup.Group{}
	for _, s := range srvs {
		s := s
		eg.Go(func() error {
			return s.MainNode().WaitReady(ctx)
		})
	}
	require.NoError(t, eg.Wait())
	return srvs
}
