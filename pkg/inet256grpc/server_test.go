package inet256grpc

import (
	"net"
	"testing"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256mem"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGRPC(t *testing.T) {
	inet256test.TestService(t, func(t testing.TB, xs []inet256.Service) {
		s := inet256mem.New()
		for i := range xs {
			srv := NewServer(s)
			gs := grpc.NewServer()
			RegisterINET256Server(gs, srv)

			l, err := net.Listen("tcp", "127.0.0.1:")
			require.NoError(t, err)
			go gs.Serve(l)
			t.Cleanup(gs.Stop)
			client, err := NewClient(l.Addr().String())
			require.NoError(t, err)
			xs[i] = client
		}
	})
}
