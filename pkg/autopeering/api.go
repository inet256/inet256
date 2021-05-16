package autopeering

import (
	"context"
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
)

type PeerStore = inet256.PeerStore

// AddrSource returns a list of addresses suitable for advertisement.
type AddrSource = func() []string

type Params struct {
	LocalAddr  inet256.Addr
	PeerStore  PeerStore
	AddrSource AddrSource
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
		logrus.Info("starting autopeering service")
		if err := srv.Run(ctx, params); err != nil {
			if err == context.Canceled {
				logrus.Infof("stopping autopeering service")
				return
			}
			logrus.Errorf("error in autopeering service %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
