package main

import (
	"context"
	"testing"
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/stretchr/testify/require"
)

func TestLocalDiscovery(t *testing.T) {
	ifaces, err := inet256d.InterfaceNames()
	require.NoError(t, err)
	// setup sides
	sides := make([]*side, 3)
	for i := range sides {
		sides[i] = newSide(t, i)
		sides[i].addDiscovery(t, inet256d.DiscoverySpec{
			Local: &inet256d.LocalDiscoverySpec{
				Interfaces: ifaces,
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
				require.Equal(t, target, inet256.NewAddr(pubKey))
			}
			return nil
		})
	}
}
