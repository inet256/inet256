package inet256http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"go.inet256.org/inet256/src/inet256"
	"go.inet256.org/inet256/src/inet256mem"
	"go.inet256.org/inet256/src/inet256tests"
)

var ctx = context.Background()

func TestOpen(t *testing.T) {
	s := NewServer(inet256mem.New())
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer l.Close()
	eg := errgroup.Group{}
	eg.Go(func() error {
		err := http.Serve(l, s)
		if errors.Is(err, net.ErrClosed) {
			err = nil
		}
		return err
	})
	eg.Go(func() error {
		defer l.Close()
		endpoint := "tcp://" + l.Addr().String()
		t.Log(endpoint)
		c, err := NewClient(endpoint)
		if err != nil {
			return err
		}
		node, err := c.Open(ctx, inet256tests.NewPrivateKey(t, 0))
		if err != nil {
			return err
		}
		return node.Close()
	})
	require.NoError(t, eg.Wait())
}

func TestService(t *testing.T) {
	inet256tests.TestService(t, func(t testing.TB, xs []inet256.Service) {
		x := inet256mem.New()
		for i := range xs {
			xs[i] = newTestService(t, x)
		}
	})
}

func BenchmarkService(b *testing.B) {
	inet256tests.BenchService(b, func(t testing.TB, xs []inet256.Service) {
		x := inet256mem.New()
		for i := range xs {
			xs[i] = newTestService(t, x)
		}
	})
}

func newTestService(t testing.TB, x inet256.Service) inet256.Service {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	t.Log("listening on", l.Addr())
	t.Cleanup(func() { l.Close() })
	go func() {
		s := NewServer(x)
		if err := http.Serve(l, s); !errors.Is(err, net.ErrClosed) {
			t.Log(err)
		}
	}()
	c, err := NewClient("tcp://" + l.Addr().String() + "/")
	require.NoError(t, err)
	return c
}
