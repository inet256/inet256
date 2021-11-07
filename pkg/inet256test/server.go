package inet256test

import (
	"context"
	"testing"

	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T, sf func(testing.TB, []inet256.Service)) {
	t.Run("Send", func(t *testing.T) {
		xs := make([]inet256.Service, 1)
		sf(t, xs)
		testServerSend(t, xs[0])
	})
	t.Run("SendMultiple", func(t *testing.T) {
		xs := make([]inet256.Service, 2)
		sf(t, xs)
		testMultipleServers(t, xs...)
	})
}

func testServerSend(t *testing.T, s inet256.Service) {
	ctx := context.Background()
	const N = 5
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := p2ptest.NewTestKey(t, i)
		n, err := s.Open(ctx, pk)
		require.NoError(t, err)
		nodes[i] = n
	}
	t.Log("created", N, "nodes")
	randomPairs(len(nodes), func(i, j int) {
		TestSendRecvOne(t, nodes[i], nodes[j])
	})
}

func testMultipleServers(t *testing.T, srvs ...inet256.Service) {
	ctx := context.Background()
	const N = 5
	nodes := make([]inet256.Node, len(srvs)*N)
	for i, s := range srvs {
		for j := 0; j < N; j++ {
			pk := p2ptest.NewTestKey(t, i*N+j)
			n, err := s.Open(ctx, pk)
			require.NoError(t, err)
			nodes[i*N+j] = n
		}
	}
	t.Log("created", N, "nodes", "on each of", len(srvs), "servers")
	randomPairs(len(nodes), func(i, j int) {
		TestSendRecvOne(t, nodes[i], nodes[j])
	})
}
