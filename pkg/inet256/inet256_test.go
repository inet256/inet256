package inet256_test

import (
	"context"
	"testing"

	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/stretchr/testify/require"
)

func TestServerLoopback(t *testing.T) {
	s := inet256test.NewTestServer(t, inet256.OneHopFactory)
	mainNode := s.MainNode()
	inet256test.TestSendRecvOne(t, mainNode, mainNode)

	const N = 5
	ctx := context.Background()
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := p2ptest.NewTestKey(t, i)
		var err error
		nodes[i], err = s.CreateNode(ctx, pk)
		require.NoError(t, err)
	}
	for i := range nodes {
		inet256test.TestSendRecvOne(t, nodes[i], nodes[i])
	}
}

func TestServerOneHop(t *testing.T) {
	s := inet256test.NewTestServer(t, inet256.OneHopFactory)
	main := s.MainNode()

	const N = 5
	ctx := context.Background()
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := p2ptest.NewTestKey(t, i)
		var err error
		nodes[i], err = s.CreateNode(ctx, pk)
		require.NoError(t, err)
	}
	for i := range nodes {
		inet256test.TestSendRecvOne(t, main, nodes[i])
	}
	for i := range nodes {
		inet256test.TestSendRecvOne(t, nodes[i], main)
	}
}
