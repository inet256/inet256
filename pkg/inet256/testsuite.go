package inet256

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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestSuite(t *testing.T, nf NetworkFactory) {
	t.Run("Test2Nodes", func(t *testing.T) {
		const N = 2
		adjList := p2ptest.Chain(N)
		nets := setupNetworks(t, N, adjList, nf)
		for _, i := range rand.Perm(len(nets)) {
			for j := range rand.Perm(len(nets)) {
				if i != j {
					TestSendRecv(t, nets[i], nets[j])
				}
			}
		}
	})
	t.Run("Test10Nodes", func(t *testing.T) {
		const N = 10
		adjList := p2ptest.Ring(N)
		nets := setupNetworks(t, N, adjList, nf)
		for _, i := range rand.Perm(len(nets)) {
			eg := errgroup.Group{}
			for j := range rand.Perm(len(nets)) {
				j := j
				if i == j {
					continue
				}
				eg.Go(func() error {
					TestSendRecv(t, nets[i], nets[j])
					return nil
				})
			}
			require.NoError(t, eg.Wait())
		}
	})
}

func TestSendRecv(t testing.TB, from, to Network) {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	actualData := "test data"
	ch := make(chan struct{}, 1)
	to.OnRecv(func(src, dst Addr, data []byte) {
		require.Equal(t, string(data), actualData)
		ch <- struct{}{}
	})
	err := from.Tell(ctx, to.LocalAddr(), []byte(actualData))
	require.NoError(t, err)
	select {
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	case <-ch:
	}
}

func setupNetworks(t *testing.T, N int, adjList p2ptest.AdjList, nf NetworkFactory) []Network {
	r := memswarm.NewRealm()

	swarms := make([]p2p.SecureSwarm, N)
	peerSwarms := make([]peerswarm.Swarm, N)
	peerStores := make([]PeerStore, N)
	keys := make([]p2p.PrivateKey, N)

	for i := 0; i < N; i++ {
		k := p2ptest.NewTestKey(t, i)
		keys[i] = k
		swarms[i] = r.NewSwarmWithKey(k)
		peerStores[i] = NewPeerStore()
		peerSwarms[i] = peerswarm.NewSwarm(swarms[i], newAddrSource(swarms[i], peerStores[i]))
	}

	for i := 0; i < N; i++ {
		for _, j := range adjList[i] {
			peerID := p2p.NewPeerID(swarms[j].PublicKey())
			addr := swarms[j].LocalAddrs()[0]
			peerStores[i].Add(peerID)
			peerStores[i].SetAddrs(peerID, []string{addr.Key()})
		}
	}
	nets := make([]Network, N)
	for i := 0; i < N; i++ {
		logger := logrus.New()
		logger.Level = logrus.DebugLevel
		logger.SetOutput(os.Stderr)
		logger.SetFormatter(&logrus.TextFormatter{
			ForceColors: true,
		})
		nets[i] = nf(NetworkParams{
			Peers:      peerStores[i],
			PrivateKey: keys[i],
			Swarm:      peerSwarms[i],
			Logger:     logger,
		})
	}

	// call WaitInit
	ctx, cf := context.WithTimeout(context.Background(), 10*time.Second)
	defer cf()
	for i := 0; i < N; i++ {
		if n, ok := nets[i].(WaitInit); ok {
			require.NoError(t, n.WaitInit(ctx))
		}
		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err(), "timeout waiting for init")
		default:
		}
	}
	t.Log("successfully initialized", N, "networks")
	t.Cleanup(func() {
		for _, n := range nets {
			require.NoError(t, n.Close())
		}
	})
	return nets
}
