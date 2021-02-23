package inet256test

import (
	"context"
	"math"
	"testing"

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
	for i := range nodes {
		for j := range nodes {
			TestSendRecvOne(t, nodes[i], nodes[j])
		}
	}
}

func newTestServer(t *testing.T, nf inet256.NetworkFactory) *inet256.Server {
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
	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})
	return s
}
