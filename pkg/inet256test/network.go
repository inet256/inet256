package inet256test

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

type (
	Addr           = inet256.Addr
	Network        = inet256.Network
	NetworkFactory = inet256.NetworkFactory
	PeerStore      = inet256.PeerStore
)

func TestNetwork(t *testing.T, nf NetworkFactory) {
	tcs := []struct {
		Name     string
		Topology p2ptest.AdjList
	}{
		{
			Name:     "Chain-2",
			Topology: p2ptest.Chain(2),
		},
		{
			Name:     "Chain-5",
			Topology: p2ptest.Chain(5),
		},
		{
			Name:     "Ring-10",
			Topology: p2ptest.Ring(10),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			nets := SetupNetworks(t, tc.Topology, nf)
			t.Log("topology:", tc.Topology)
			randomPairs(len(nets), func(i, j int) {
				TestSendRecvOne(t, nets[i], nets[j])
			})
		})
	}
}

func TestSendRecvOne(t testing.TB, src, dst Network) {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()

	eg := errgroup.Group{}
	var recieved string
	eg.Go(func() error {
		var srcAddr, dstAddr inet256.Addr
		buf := make([]byte, inet256.TransportMTU)
		n, err := dst.Recv(ctx, &srcAddr, &dstAddr, buf)
		if err != nil {
			return err
		}
		recieved = string(buf[:n])
		return nil
	})
	sent := "test data"
	eg.Go(func() error {
		return src.Tell(ctx, dst.LocalAddr(), []byte(sent))
	})
	require.NoError(t, eg.Wait())
	require.Equal(t, sent, recieved)
}

func SetupNetworks(t testing.TB, adjList p2ptest.AdjList, nf NetworkFactory) []Network {
	r := memswarm.NewRealm()
	N := len(adjList)
	swarms := make([]p2p.SecureSwarm, N)
	peerSwarms := make([]peerswarm.Swarm, N)
	peerStores := make([]PeerStore, N)
	keys := make([]p2p.PrivateKey, N)

	for i := 0; i < N; i++ {
		keys[i] = p2ptest.NewTestKey(t, i)
		swarms[i] = r.NewSwarmWithKey(keys[i])
		peerStores[i] = inet256.NewPeerStore()
		peerSwarms[i] = inet256.NewPeerSwarm(swarms[i], peerStores[i])
	}

	for i := range adjList {
		for _, j := range adjList[i] {
			peerID := p2p.NewPeerID(swarms[j].PublicKey())
			addr := swarms[j].LocalAddrs()[0]
			peerStores[i].Add(peerID)
			peerStores[i].SetAddrs(peerID, []string{addr.Key()})
		}
	}
	nets := make([]Network, N)
	for i := 0; i < N; i++ {
		logger := newTestLogger(t)
		nets[i] = nf(inet256.NetworkParams{
			Peers:      peerStores[i],
			PrivateKey: keys[i],
			Swarm:      peerSwarms[i],
			Logger:     logger,
		})
	}
	bootstrapNetworks(t, nets)
	t.Log("successfully initialized", N, "networks")
	t.Cleanup(func() {
		for _, n := range nets {
			require.NoError(t, n.Close())
		}
	})
	return nets
}

func bootstrapNetworks(t testing.TB, nets []Network) {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	for i := 0; i < len(nets); i++ {
		err := nets[i].Bootstrap(ctx)
		require.NoError(t, err)
		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err(), "timeout waiting for bootstrap")
		default:
		}
	}
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
