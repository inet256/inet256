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
			nets := setupNetworks(t, tc.Topology, nf)
			chans := setupChans(nets)
			t.Log("topology:", tc.Topology)
			randomPairs(len(nets), func(i, j int) {
				testSendRecvOne(t, nets[i], nets[j].LocalAddr(), chans[j])
			})
		})
	}
}

func testSendRecvOne(t testing.TB, from Network, dst Addr, queue chan Message) {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	actualData := "test data"

	err := from.Tell(ctx, dst, []byte(actualData))
	require.NoError(t, err)

	select {
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	case msg := <-queue:
		require.Equal(t, actualData, string(msg.Payload))
	}
	drain(queue)
}

func setupNetworks(t *testing.T, adjList p2ptest.AdjList, nf NetworkFactory) []Network {
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
		peerSwarms[i] = peerswarm.NewSwarm(swarms[i], inet256.NewAddrSource(swarms[i], peerStores[i]))
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
	waitReadyNetworks(t, nets)
	t.Log("successfully initialized", N, "networks")
	t.Cleanup(func() {
		for _, n := range nets {
			require.NoError(t, n.Close())
		}
	})
	return nets
}

func waitReadyNetworks(t *testing.T, nets []Network) {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()
	for i := 0; i < len(nets); i++ {
		err := nets[i].WaitReady(ctx)
		require.NoError(t, err)
		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err(), "timeout waiting for init")
		default:
		}
	}
}

func setupChan(nwk Network) chan Message {
	ch := make(chan Message, 10)
	go func() {
		err := nwk.Recv(func(src, dst inet256.Addr, data []byte) {
			ch <- Message{
				Src:     src,
				Dst:     dst,
				Payload: append([]byte{}, data...),
			}
		})
		close(ch)
		if err != p2p.ErrSwarmClosed {
			logrus.Error(err)
		}
	}()
	return ch
}

func setupChans(nets []Network) []chan Message {
	chans := make([]chan Message, len(nets))
	for i := range nets {
		chans[i] = setupChan(nets[i])
	}
	return chans
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

func drain(x chan Message) {
	select {
	case <-x:
	default:
		return
	}
}

func newTestLogger(t *testing.T) *logrus.Logger {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	logger.SetOutput(os.Stderr)
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
	return logger
}

type Message struct {
	Src     inet256.Addr
	Dst     inet256.Addr
	Payload []byte
}

func castNodeSlice(nodes []inet256.Node) []Network {
	nets := make([]Network, len(nodes))
	for i := range nodes {
		nets[i] = nodes[i]
	}
	return nets
}
