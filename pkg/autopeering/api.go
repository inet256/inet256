package autopeering

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/stdctx/logctx"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/peers"
)

type TransportAddr = multiswarm.Addr

// AddrSource returns a list of addresses suitable for advertisement.
type AddrSource = func() []TransportAddr

type Params struct {
	// Outbound
	LocalAddr  inet256.Addr
	AddrSource AddrSource

	// Inbound
	PeerStore peers.Store[TransportAddr]
	ParseAddr func([]byte) (TransportAddr, error)
}

// Service manages the peers in a PeerStore, adding to and removing from them automatically
// as peers are discovered and lost.
type Service interface {
	// Run should run until the context is cancelled, calling params.AddrSource to
	// get the local addresses, and storing discovered addresses in params.PeerStore
	Run(ctx context.Context, params Params) error
}

func RunForever(ctx context.Context, srv Service, params Params) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		logctx.Infoln(ctx, "starting autopeering service")
		if err := srv.Run(ctx, params); err != nil {
			if err == context.Canceled {
				logctx.Infoln(ctx, "stopping autopeering service")
				return
			}
			logctx.Errorf(ctx, "error in autopeering service %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
