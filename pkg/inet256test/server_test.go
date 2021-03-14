package inet256test

import (
	"context"
	"testing"

	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
)

type Node = inet256.Node

func TestLoopback(t *testing.T) {
	s := newTestServer(t, inet256.OneHopFactory)
	mainNode := s.MainNode()
	mainChan := setupChan(mainNode)
	testSendRecvOne(t, mainNode, mainNode.LocalAddr(), mainChan)

	const N = 5
	ctx := context.Background()
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := p2ptest.NewTestKey(t, i)
		var err error
		nodes[i], err = s.CreateNode(ctx, pk)
		require.NoError(t, err)
	}
	chans := setupChans(castNodeSlice(nodes))
	for i := range nodes {
		testSendRecvOne(t, nodes[i], nodes[i].LocalAddr(), chans[i])
	}
}

func TestServerOneHop(t *testing.T) {
	s := newTestServer(t, inet256.OneHopFactory)
	main := s.MainNode()
	mainChan := setupChan(main)

	const N = 5
	ctx := context.Background()
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := p2ptest.NewTestKey(t, i)
		var err error
		nodes[i], err = s.CreateNode(ctx, pk)
		require.NoError(t, err)
	}
	chans := setupChans(castNodeSlice(nodes))
	for i := range nodes {
		testSendRecvOne(t, main, nodes[i].LocalAddr(), chans[i])
	}
	for i := range nodes {
		testSendRecvOne(t, nodes[i], main.LocalAddr(), mainChan)
	}
}
