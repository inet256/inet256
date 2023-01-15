package mesh256_test

import (
	"context"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/stretchr/testify/require"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/inet256/inet256/pkg/mesh256"
)

func TestServerLoopback(t *testing.T) {
	s := mesh256.NewTestServer(t, oneHopFactory)
	mainNode := s.MainNode()
	inet256test.TestSendRecvOne(t, mainNode, mainNode)

	const N = 5
	ctx := context.Background()
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := inet256test.NewPrivateKey(t, i)
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
		pk := inet256test.NewPrivateKey(t, i)
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

func TestServerCreateDrop(t *testing.T) {
	ctx := context.Background()
	s := mesh256.NewTestServer(t, oneHopFactory)

	const N = 100
	for i := 0; i < N; i++ {
		pk := inet256test.NewPrivateKey(t, i)
		_, err := s.Open(ctx, pk)
		require.NoError(t, err)
	}

	for i := 0; i < N; i++ {
		pk := inet256test.NewPrivateKey(t, i)
		err := s.Drop(ctx, pk)
		require.NoError(t, err)
	}
}

func oneHopFactory(params mesh256.NetworkParams) mesh256.Network {
	return oneHop{params}
}

var _ mesh256.Network = oneHop{}

type oneHop struct {
	params mesh256.NetworkParams
}

func (n oneHop) LocalAddr() inet256.Addr {
	return inet256.NewAddr(n.params.PrivateKey.Public())
}

func (n oneHop) MTU(ctx context.Context, target inet256.Addr) int {
	return n.params.Swarm.MTU(ctx, target)
}

func (n oneHop) Close() error {
	return nil
}

func (n oneHop) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	return n.params.Swarm.LookupPublicKey(ctx, target)
}

func (n oneHop) PublicKey() inet256.PublicKey {
	return n.params.PrivateKey.Public()
}

func (n oneHop) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	for _, id := range n.params.Peers.ListPeers() {
		if inet256.HasPrefix(id[:], prefix, nbits) {
			return inet256.Addr(id), nil
		}
	}
	return inet256.Addr{}, inet256.ErrNoAddrWithPrefix
}

func (n oneHop) Tell(ctx context.Context, dst inet256.Addr, v p2p.IOVec) error {
	if !n.params.Peers.Contains(dst) {
		return inet256.ErrAddrUnreachable{Addr: dst}
	}
	return n.params.Swarm.Tell(ctx, dst, v)
}

func (n oneHop) Receive(ctx context.Context, fn func(p2p.Message[inet256.Addr])) error {
	return n.params.Swarm.Receive(ctx, fn)
}
