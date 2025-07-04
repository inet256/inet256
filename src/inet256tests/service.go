package inet256tests

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/cloudflare/circl/sign/ed25519"
	"github.com/stretchr/testify/require"
	"go.brendoncarroll.net/p2p/p2ptest"
	"golang.org/x/sync/errgroup"

	"go.inet256.org/inet256/src/inet256"
)

var ctx = context.Background()

func TestService(t *testing.T, sf func(testing.TB, []inet256.Service)) {
	t.Run("Send", func(t *testing.T) {
		t.Parallel()
		xs := make([]inet256.Service, 1)
		sf(t, xs)
		testServerSend(t, xs[0])
	})
	t.Run("SendMultiple", func(t *testing.T) {
		t.Parallel()
		xs := make([]inet256.Service, 2)
		sf(t, xs)
		testMultipleServers(t, xs...)
	})
	t.Run("TestLoopback", func(t *testing.T) {
		t.Parallel()
		xs := make([]inet256.Service, 2)
		sf(t, xs)
		n1 := OpenNode(t, xs[0], 1)
		n2 := OpenNode(t, xs[1], 2)
		TestSendRecvOne(t, n1, n1)
		TestSendRecvOne(t, n2, n2)
	})
	t.Run("FindAddr", func(t *testing.T) {
		t.Parallel()
		xs := make([]inet256.Service, 2)
		sf(t, xs)
		n1 := OpenNode(t, xs[0], 1)
		n2 := OpenNode(t, xs[1], 2)
		n2Addr := n2.LocalAddr()
		addr, err := n1.FindAddr(ctx, n2Addr[:1], 7)
		require.NoError(t, err)
		require.Equal(t, addr, n2Addr)
	})
	t.Run("LookupPublicKey", func(t *testing.T) {
		t.Parallel()
		xs := make([]inet256.Service, 2)
		sf(t, xs)
		n1 := OpenNode(t, xs[0], 1)
		n2 := OpenNode(t, xs[1], 2)
		pubKey, err := n1.LookupPublicKey(ctx, n2.LocalAddr())
		require.NoError(t, err)
		require.Equal(t, inet256.MarshalPublicKey(nil, n2.PublicKey()), inet256.MarshalPublicKey(nil, pubKey))
	})
	t.Run("ReOpen", func(t *testing.T) {
		t.Parallel()
		xs := make([]inet256.Service, 1)
		sf(t, xs)
		x := xs[0]
		priv := NewPrivateKey(t, 1)
		// Check close
		n, err := x.Open(ctx, priv)
		require.NoError(t, err)
		require.NotNil(t, n)
		require.NoError(t, n.Close())
		// Check Drop
		n, err = x.Open(ctx, priv)
		require.NoError(t, err)
		require.NotNil(t, n)
		require.NoError(t, x.Drop(ctx, priv))

		n = OpenNode(t, xs[0], 1)
		require.NotNil(t, n)
	})
}

func testServerSend(t *testing.T, s inet256.Service) {
	ctx := context.Background()
	const N = 5
	nodes := make([]inet256.Node, N)
	for i := range nodes {
		pk := NewPrivateKey(t, i)
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
			pk := NewPrivateKey(t, i*N+j)
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

func TestSendRecvOne(t testing.TB, src, dst inet256.Node) {
	ctx, cf := context.WithTimeout(context.Background(), 2*time.Second)
	defer cf()

	eg := errgroup.Group{}
	var recieved inet256.Message
	eg.Go(func() error {
		if err := inet256.Receive(ctx, dst, &recieved); err != nil {
			return err
		}
		return nil
	})
	sent := "test data"
	eg.Go(func() error {
		return src.Send(ctx, dst.LocalAddr(), []byte(sent))
	})
	require.NoError(t, eg.Wait())
	require.Equal(t, sent, string(recieved.Payload))
	// require.Equal(t, src.LocalAddr(), recieved.Src)
	require.Equal(t, dst.LocalAddr(), recieved.Dst)
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

// NewPrivateKey creates an insecure, but deterministic, and easy to recreate private key suitable for tests.
func NewPrivateKey(t testing.TB, i int) inet256.PrivateKey {
	pk := p2ptest.NewTestKey(t, i)
	return ed25519.PrivateKey(pk)
}

func OpenNode(t testing.TB, s inet256.Service, i int) inet256.Node {
	ctx := context.Background()
	pk := NewPrivateKey(t, i)
	n, err := s.Open(ctx, pk)
	require.NoError(t, err)
	t.Cleanup(func() {
		n.Close()
	})
	return n
}
