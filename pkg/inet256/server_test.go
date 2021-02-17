package inet256

import (
	"math"
	"testing"

	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	s := newTestServer(t)

	const N = 10
	nodes := make([]Node, N)
	for i := 0; i < N; i++ {
		pk := p2ptest.NewTestKey(t, i+1)
		n, err := s.CreateNode(pk)
		require.Nil(t, err)
		nodes[i] = n
		t.Log(n.LocalAddr())
	}

	// TODO: TestSendRecv
}

func newTestServer(t *testing.T) *Server {
	pk := p2ptest.NewTestKey(t, math.MaxInt32)
	ps := NewPeerStore()
	s := NewServer(Params{
		Networks:   []NetworkSpec{},
		Peers:      ps,
		PrivateKey: pk,
	})
	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})
	return s
}
