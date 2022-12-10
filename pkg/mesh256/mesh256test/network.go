package mesh256test

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/peers"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type (
	Addr           = inet256.Addr
	Node           = inet256.Node
	Network        = mesh256.Network
	NetworkFactory = mesh256.NetworkFactory
)

func TestNetwork(t *testing.T, nf NetworkFactory) {
	tcs := []struct {
		Name     string
		Topology p2ptest.AdjList
	}{
		{
			Name:     "Chain-2",
			Topology: p2ptest.MakeChain(2),
		},
		{
			Name:     "Chain-5",
			Topology: p2ptest.MakeChain(5),
		},
		{
			Name:     "Ring-10",
			Topology: p2ptest.MakeRing(10),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			nets := SetupNetworks(t, tc.Topology, nf)
			t.Log("topology:", tc.Topology)
			t.Run("FindAddr", func(t *testing.T) {
				randomPairs(len(nets), func(i, j int) {
					TestFindAddr(t, nets[i], nets[j])
				})
			})
			t.Run("SendRecvAll", func(t *testing.T) {
				TestSendRecvAll(t, nets)
			})
		})
	}
}

func TestFindAddr(t testing.TB, src, dst Network) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second)
	defer cf()
	dstAddr := dst.LocalAddr()
	src.FindAddr(ctx, dstAddr[:], len(dstAddr)/8)
}

func TestSendRecvAll(t testing.TB, nets []Network) {
	randomPairs(len(nets), func(i, j int) {
		inet256test.TestSendRecvOne(t, net2node{nets[i]}, net2node{nets[j]})
	})
}

func SetupNetworks(t testing.TB, adjList p2ptest.AdjList, nf NetworkFactory) []Network {
	N := len(adjList)
	swarms := make([]*memswarm.Swarm, N)
	peerStores := make([]peers.Store[memswarm.Addr], N)
	keys := make([]inet256.PrivateKey, N)
	netSwarms := make([]mesh256.Swarm, N)
	r := memswarm.NewRealm()
	for i := 0; i < N; i++ {
		keys[i] = inet256test.NewPrivateKey(t, i)
		swarms[i] = r.NewSwarmWithKey(keys[i].BuiltIn())
		peerStores[i] = peers.NewStore[memswarm.Addr]()
		netSwarms[i] = mesh256.NewSwarm[memswarm.Addr](swarms[i], peerStores[i])
	}

	for i := range adjList {
		for _, j := range adjList[i] {
			peerID := inet256.NewAddr(netSwarms[j].PublicKey())
			addr := swarms[j].LocalAddrs()[0]
			peerStores[i].Add(peerID)
			peerStores[i].SetAddrs(peerID, []memswarm.Addr{addr})
		}
	}
	nets := make([]Network, N)
	for i := 0; i < N; i++ {
		logger := newTestLogger(t)
		nets[i] = nf(mesh256.NetworkParams{
			Peers:      peerStores[i],
			PrivateKey: keys[i],
			Swarm:      netSwarms[i],
			Logger:     logger,
		})
	}
	t.Log("successfully initialized", N, "networks")
	t.Cleanup(func() {
		for _, n := range nets {
			require.NoError(t, n.Close())
		}
	})
	return nets
}

func randomPairs(n int, fn func(i, j int)) {
	for _, i := range rand.Perm(n) {
		for _, j := range rand.Perm(n) {
			if i != j {
				fn(i, j)
			}
		}
	}
}

func newTestLogger(t testing.TB) *logrus.Logger {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	logger.SetOutput(os.Stderr)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
	return logger
}

type net2node struct {
	Network
}

func (x net2node) Send(ctx context.Context, dst Addr, data []byte) error {
	return x.Network.Tell(ctx, dst, p2p.IOVec{data})
}

func (x net2node) Receive(ctx context.Context, fn func(inet256.Message)) error {
	return x.Network.Receive(ctx, func(x p2p.Message[Addr]) {
		fn(inet256.Message{
			Src:     x.Src,
			Dst:     x.Dst,
			Payload: x.Payload,
		})
	})
}
