package inet256test

import (
	"context"
	"testing"

	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
)

func TestLoopback(t *testing.T) {
	s := newTestServer(t, inet256.OneHopFactory)
	n := s.MainNode()
	TestSendRecvOne(t, n, n)

	const N = 5
	ctx := context.Background()
	nodes := make([]inet256.Node, 5)
	for i := range nodes {
		pk := p2ptest.NewTestKey(t, i)
		var err error
		nodes[i], err = s.CreateNode(ctx, pk)
		require.NoError(t, err)
	}
	for i := range nodes {
		TestSendRecvOne(t, nodes[i], nodes[i])
	}
}

// func TestOneHop(t *testing.T) {
// 	s := newTestServer(t, inet256.OneHopFactory)
// 	main := s.MainNode()

// 	const N = 5
// 	ctx := context.Background()
// 	nodes := make([]inet256.Node, 5)
// 	for i := range nodes {
// 		pk := p2ptest.NewTestKey(t, i)
// 		var err error
// 		nodes[i], err = s.CreateNode(ctx, pk)
// 		require.NoError(t, err)
// 	}
// 	for i := range nodes {
// 		TestSendRecvOne(t, main, nodes[i])
// 	}
// 	for i := range nodes {
// 		TestSendRecvOne(t, nodes[i], main)
// 	}
// }
