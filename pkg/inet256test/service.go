package inet256test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
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
