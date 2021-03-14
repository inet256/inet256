package inet256test

import (
	"context"
	"log"
	"math"
	"testing"
	"time"

	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T, nf inet256.NetworkFactory) {
	ctx := context.Background()
	s := newTestServer(t, nf)
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
	log.Println(chans)
	randomPairs(len(nodes), func(i, j int) {
		log.Println("sending", nodes[i].LocalAddr(), nodes[j].LocalAddr())
		log.Println("sending to chan", chans[j])
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
