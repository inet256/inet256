package centraldisco

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.brendoncarroll.net/p2p/s/multiswarm"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"go.inet256.org/inet256/pkg/discovery/centraldisco/internal"
	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/inet256test"
)

func TestClientServer(t *testing.T) {
	ctx := context.Background()
	// server
	s := NewServer(addrParser)
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer l.Close()
	gs := grpc.NewServer()
	internal.RegisterDiscoveryServer(gs, s)

	// clients
	gc, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)
	c1 := NewClient(gc)
	c2 := NewClient(gc)
	pk1 := inet256test.NewPrivateKey(t, 0)
	pk2 := inet256test.NewPrivateKey(t, 1)

	eg := errgroup.Group{}
	eg.Go(func() error {
		return gs.Serve(l)
	})

	// announce
	err = c1.Announce(ctx, pk1, []string{"udp://127.0.0.1:1234"}, time.Hour)
	require.NoError(t, err)
	err = c2.Announce(ctx, pk2, []string{"udp://127.0.0.1:1235"}, time.Hour)
	require.NoError(t, err)
	// find
	endpoints, err := c1.Lookup(ctx, inet256.NewAddr(pk1.Public()))
	require.NoError(t, err)
	require.Len(t, endpoints, 1)
	endpoints, err = c2.Lookup(ctx, inet256.NewAddr(pk2.Public()))
	require.NoError(t, err)
	require.Len(t, endpoints, 1)

	gs.Stop()
	require.NoError(t, eg.Wait())
}

func addrParser(x []byte) (multiswarm.Addr, error) {
	if !bytes.HasPrefix(x, []byte("udp://")) {
		return multiswarm.Addr{}, errors.New("not udp address")
	}
	return multiswarm.Addr{}, nil
}
