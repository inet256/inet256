package discovery

import (
	"context"

	"go.brendoncarroll.net/p2p"
	"go.brendoncarroll.net/p2p/s/multiswarm"
	"go.brendoncarroll.net/stdctx/logctx"

	"go.inet256.org/inet256/internal/retry"
	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/peers"
)

type (
	TransportAddr = multiswarm.Addr
	AddrParser    = p2p.AddrParser[TransportAddr]
	AddrSource    = func() []TransportAddr
)

// AddressBook is allows storing addresses for a Peer
type AddressBook interface {
	peers.Updater[TransportAddr]
	List() []inet256.Addr
}

type Params struct {
	AutoPeering bool

	// Announcing
	PrivateKey    inet256.PrivateKey
	LocalID       inet256.Addr
	GetLocalAddrs func() []TransportAddr

	// Finding
	Peers      peers.Store[TransportAddr]
	AddrParser p2p.AddrParser[TransportAddr]
}

// AddrService find transport addresses for known peers
type Service interface {
	Run(ctx context.Context, params Params) error
}

func RunForever(ctx context.Context, srv Service, params Params) {
	logctx.Infoln(ctx, "starting address discovery service")
	retry.Retry(ctx, func() error {
		return srv.Run(ctx, params)
	}, retry.WithPredicate(func(err error) bool {
		logctx.Errorln(ctx, "error in address discovery service", err)
		return true
	}))
	logctx.Infoln(ctx, "stopped address discovery service")
}

func DisableAddRemove[T p2p.Addr](x peers.Store[T]) peers.Store[T] {
	return fixedStore[T]{x}
}

type fixedStore[T p2p.Addr] struct {
	peers.Store[T]
}

func (s fixedStore[T]) Add(x inet256.Addr) {}

func (s fixedStore[T]) Remove(x inet256.Addr) {}
