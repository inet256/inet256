package discovery

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/stdctx/logctx"

	"github.com/inet256/inet256/internal/retry"
	"github.com/inet256/inet256/pkg/inet256"
)

type (
	TransportAddr = multiswarm.Addr
	AddrParser    = p2p.AddrParser[TransportAddr]
)

type AddressBook interface {
	SetAddrs(x inet256.Addr, addrs []TransportAddr)
	ListPeers() []inet256.Addr
}

type Params struct {
	// Announcing
	PrivateKey    inet256.PrivateKey
	LocalID       inet256.Addr
	GetLocalAddrs func() []TransportAddr

	// Finding
	AddressBook AddressBook
	AddrParser  p2p.AddrParser[TransportAddr]
}

type Service interface {
	Run(ctx context.Context, params Params) error
}

func RunForever(ctx context.Context, srv Service, params Params) {
	logctx.Infoln(ctx, "starting discovery service")
	retry.Retry(ctx, func() error {
		return srv.Run(ctx, params)
	}, retry.WithPredicate(func(err error) bool {
		logctx.Errorln(ctx, "error in discovery service", err)
		return true
	}))
	logctx.Infoln(ctx, "stopped discovery service")
}
