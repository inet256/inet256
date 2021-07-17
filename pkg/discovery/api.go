package discovery

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/inet256/inet256/pkg/serde"
)

type AddressBook interface {
	SetAddrs(x inet256.Addr, addrs []p2p.Addr)
	ListPeers() []inet256.Addr
}

type Params struct {
	// Announcing
	LocalID       inet256.Addr
	GetLocalAddrs func() []p2p.Addr

	// Finding
	AddressBook AddressBook
	AddrParser  serde.AddrParserFunc

	Logger *inet256.Logger
}

type Service interface {
	Run(ctx context.Context, params Params) error
}

func RunForever(ctx context.Context, srv Service, params Params) {
	params.Logger.Info("starting discovery service")
	netutil.Retry(ctx, func() error {
		return srv.Run(ctx, params)
	}, netutil.WithPredicate(func(err error) bool {
		params.Logger.WithError(err).Error()
		return true
	}))
	params.Logger.Info("stopped discovery service")
}
