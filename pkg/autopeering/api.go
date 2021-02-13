package autopeering

import (
	"context"

	"github.com/inet256/inet256/pkg/inet256"
)

type PeerStore = inet256.PeerStore

// AddrSource returns a list of addresses suitable for advertisement.
type AddrSource = func() []string

// Service manages the peers in a PeerStore, adding to and removing from them automatically
// as peers are discovered and lost.
type Service interface {
	// Run should run until the context is cancelled, calling addrSource to
	// get the local addresses, and storing discovered addresses in peerStore
	Run(ctx context.Context, localID inet256.Addr, peerStore PeerStore, addrSource AddrSource) error
}
