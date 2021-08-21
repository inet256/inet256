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
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestServer(t *testing.T, nf inet256.NetworkFactory) {
	t.Run("Send", func(t *testing.T) {
		s := NewTestServer(t, nf)
		testServerSend(t, s)
	})
	t.Run("SendMultiple", func(t *testing.T) {
		srvs := NewTestServers(t, nf, 2)
		testMultipleServers(t, srvs...)
	})
}

func testServerSend(t *testing.T, s *inet256srv.Server) {
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
	randomPairs(len(nodes), func(i, j int) {
		TestSendRecvOne(t, nodes[i], nodes[j])
	})
}

func testMultipleServers(t *testing.T, srvs ...*inet256srv.Server) {
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
	randomPairs(len(nodes), func(i, j int) {
		TestSendRecvOne(t, nodes[i], nodes[j])
	})
}

func NewTestServer(t testing.TB, nf inet256.NetworkFactory) *inet256srv.Server {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	pk := p2ptest.NewTestKey(t, math.MaxInt32)
	ps := inet256srv.NewPeerStore()
	s := inet256srv.NewServer(inet256srv.Params{
		Networks: []inet256srv.NetworkSpec{
			{
				Factory: nf,
				Index:   0,
				Name:    "network-name",
			},
		},
		Peers:      ps,
		PrivateKey: pk,
	})
	err := s.MainNode().Bootstrap(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})
	return s
}

func NewTestServers(t *testing.T, nf inet256.NetworkFactory, n int) []*inet256srv.Server {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	r := memswarm.NewRealm()
	stores := make([]inet256.PeerStore, n)
	srvs := make([]*inet256srv.Server, n)
	for i := range srvs {
		pk := p2ptest.NewTestKey(t, math.MaxInt32+i)
		stores[i] = inet256srv.NewPeerStore()
		srvs[i] = inet256srv.NewServer(inet256srv.Params{
			Swarms: map[string]p2p.SecureSwarm{
				"external": r.NewSwarmWithKey(pk),
			},
			Networks: []inet256srv.NetworkSpec{
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
			return s.MainNode().Bootstrap(ctx)
		})
	}
	require.NoError(t, eg.Wait())
	return srvs
}
