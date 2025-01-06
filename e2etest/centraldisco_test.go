package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.inet256.org/inet256/pkg/discovery"
	"go.inet256.org/inet256/pkg/discovery/centraldisco"
	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/inet256d"
	"go.inet256.org/inet256/pkg/mesh256"
	"google.golang.org/grpc"
)

func TestCentralDiscovery(t *testing.T) {
	// setup discovery server
	discoSrv := centraldisco.NewServer(func([]byte) (discovery.TransportAddr, error) {
		return discovery.TransportAddr{}, nil
	})
	centralAddr := runGRPCServer(t, func(gs *grpc.Server) {
		centraldisco.RegisterServer(gs, discoSrv)
	})
	// setup sides
	sides := make([]*side, 3)
	for i := range sides {
		sides[i] = newSide(t, i)
		sides[i].addDiscovery(t, inet256d.DiscoverySpec{
			Central: &inet256d.CentralDiscoverySpec{
				Endpoint: "http://" + centralAddr,
				Period:   1000 * time.Millisecond,
			},
		})
	}
	for i := range sides {
		for j := range sides {
			if i == j {
				continue
			}
			sides[i].addPeer(t, inet256d.PeerSpec{
				ID:    sides[j].localAddr(),
				Addrs: nil, // no transport addresses
			})
		}
	}
	for i := range sides {
		sides[i].startDaemon(t)
	}
	time.Sleep(2000 * time.Millisecond)

	ctx := context.Background()
	for i := range sides {
		sides[0].d.DoWithServer(ctx, func(s *mesh256.Server) error {
			for j := range sides {
				if i == j {
					continue
				}
				target := sides[j].localAddr()
				addr2, err := s.FindAddr(ctx, target[:], 256)
				require.NoError(t, err)
				require.Equal(t, target, addr2)

				pubKey, err := s.LookupPublicKey(ctx, target)
				require.NoError(t, err)
				require.Equal(t, target, inet256.NewAddr(pubKey))
			}
			return nil
		})
	}
}

func runGRPCServer(t testing.TB, regFn func(gs *grpc.Server)) string {
	l, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	gs := grpc.NewServer()
	regFn(gs) // register servers
	done := make(chan struct{})
	t.Cleanup(func() {
		gs.Stop()
		<-done
	})
	go func() {
		defer close(done)
		gs.Serve(l)
	}()
	return l.Addr().String()
}
