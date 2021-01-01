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
)

func TestSuite(t *testing.T, nf NetworkFactory) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second)
	defer cf()
	const N = 10
	r := memswarm.NewRealm()

	swarms := make([]p2p.SecureSwarm, N)
	peerSwarms := make([]peerswarm.Swarm, N)
	peerStores := make([]MutablePeerStore, N)
	keys := make([]p2p.PrivateKey, N)

	for i := 0; i < 10; i++ {
		k := p2ptest.NewTestKey(t, i)
		keys[i] = k
		swarms[i] = r.NewSwarmWithKey(k)
		peerStores[i] = NewPeerStore()
		peerSwarms[i] = peerswarm.NewSwarm(swarms[i], newAddrSource(swarms[i], peerStores[i]))
	}
	adjList := p2ptest.Chain(p2ptest.CastSlice(swarms))
	for i := 0; i < N; i++ {
		for _, addr := range adjList[i] {
			j := addr.(memswarm.Addr).N
			peerID := p2p.NewPeerID(swarms[j].PublicKey())
			peerStores[i].AddPeer(peerID)
			peerStores[i].PutAddrs(peerID, []string{addr.Key()})
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
		if n, ok := nets[i].(WaitInit); ok {
			require.NoError(t, n.WaitInit(ctx))
		}
		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		default:
		}
	}
	defer func() {
		for _, n := range nets {
			require.NoError(t, n.Close())
		}
	}()
	time.Sleep(time.Second)
	for _, i := range rand.Perm(N) {
		for j := range rand.Perm(N) {
			if i != j {
				TestSendRecv(t, nets[i], nets[j])
			}
		}
	}
}

func TestSendRecv(t testing.TB, from, to Network) {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	actualData := "test data"
	ch := make(chan struct{}, 1)
	to.OnRecv(func(src, dst Addr, data []byte) {
		require.Equal(t, string(data), actualData)
	})
	err := from.Tell(ctx, to.LocalAddr(), []byte(actualData))
	require.NoError(t, err)
	select {
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	case <-ch:
	}
}
