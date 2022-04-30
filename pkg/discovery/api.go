package discovery

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/retry"
	"github.com/sirupsen/logrus"
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

	Logger *logrus.Logger
}

type Service interface {
	Run(ctx context.Context, params Params) error
}

func RunForever(ctx context.Context, srv Service, params Params) {
	params.Logger.Info("starting discovery service")
	retry.Retry(ctx, func() error {
		return srv.Run(ctx, params)
	}, retry.WithPredicate(func(err error) bool {
		params.Logger.WithError(err).Error()
		return true
	}))
	params.Logger.Info("stopped discovery service")
}
