package mesh256_test

import (
	"context"
	"testing"

	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/stretchr/testify/require"
)

func TestServerLoopback(t *testing.T) {
	s := mesh256.NewTestServer(t, oneHopFactory)
	mainNode := s.MainNode()
	inet256test.TestSendRecvOne(t, mainNode, mainNode)

	const N = 5
	ctx := context.Background()
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := p2ptest.NewTestKey(t, i)
		var err error
		nodes[i], err = s.Open(ctx, pk)
		require.NoError(t, err)
	}
	for i := range nodes {
		inet256test.TestSendRecvOne(t, nodes[i], nodes[i])
	}
}

func TestServerOneHop(t *testing.T) {
	s := mesh256.NewTestServer(t, oneHopFactory)
	main := s.MainNode()

	const N = 5
	ctx := context.Background()
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := p2ptest.NewTestKey(t, i)
		var err error
		nodes[i], err = s.Open(ctx, pk)
		require.NoError(t, err)
	}
	for i := range nodes {
		inet256test.TestSendRecvOne(t, main, nodes[i])
	}
	for i := range nodes {
		inet256test.TestSendRecvOne(t, nodes[i], main)
	}
}

func TestServerCreateDelete(t *testing.T) {
	ctx := context.Background()
	s := mesh256.NewTestServer(t, oneHopFactory)

	const N = 100
	for i := 0; i < N; i++ {
		pk := p2ptest.NewTestKey(t, i)
		_, err := s.Open(ctx, pk)
		require.NoError(t, err)
	}

	for i := 0; i < N; i++ {
		pk := p2ptest.NewTestKey(t, i)
		err := s.Delete(ctx, pk)
		require.NoError(t, err)
	}
}

func oneHopFactory(params mesh256.NetworkParams) mesh256.Network {
	findAddr := func(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
		for _, id := range params.Peers.ListPeers() {
			if inet256.HasPrefix(id[:], prefix, nbits) {
				return inet256.Addr(id), nil
			}
		}
		return inet256.Addr{}, inet256.ErrNoAddrWithPrefix
	}
	waitReady := func(ctx context.Context) error {
		return nil
	}
	return mesh256.NetworkFromSwarm(params.Swarm, findAddr, waitReady)
}
