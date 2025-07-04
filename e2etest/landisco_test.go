package main

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.inet256.org/inet256/src/inet256"
	"go.inet256.org/inet256/src/inet256d"
	"go.inet256.org/inet256/src/mesh256"
)

func TestLocalDiscovery(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skipf("local discovery broken on darwin")
	}
	ifaces, err := inet256d.InterfaceNames()
	require.NoError(t, err)
	// setup sides
	sides := make([]*side, 3)
	for i := range sides {
		sides[i] = newSide(t, i)
		sides[i].addDiscovery(t, inet256d.DiscoverySpec{
			Local: &inet256d.LocalDiscoverySpec{
				Interfaces:     ifaces,
				AnnouncePeriod: time.Second,
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
		sides[i].d.DoWithServer(ctx, func(s *mesh256.Server) error {
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
				require.Equal(t, target, inet256.NewID(pubKey))
			}
			return nil
		})
	}
}
