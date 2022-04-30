package inet256grpc

import (
	"net"
	"testing"

	"github.com/inet256/inet256/networks/beaconnet"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGRPC(t *testing.T) {
	inet256test.TestService(t, func(t testing.TB, xs []inet256.Service) {
		mesh256.NewTestServers(t, beaconnet.Factory, xs)
		for i := range xs {
			srv := NewServer(xs[i])
			gs := grpc.NewServer()
			RegisterINET256Server(gs, srv)

			l, err := net.Listen("tcp", "127.0.0.1:")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, l.Close()) })
			go gs.Serve(l)
			//t.Cleanup(gs.Stop)
			client, err := NewClient(l.Addr().String())
			require.NoError(t, err)
			xs[i] = client
		}
	})
}
