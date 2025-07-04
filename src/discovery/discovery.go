package discovery

import (
	"context"

	"go.brendoncarroll.net/p2p"
	"go.brendoncarroll.net/p2p/s/multiswarm"
	"go.brendoncarroll.net/stdctx/logctx"

	"go.inet256.org/inet256/src/inet256"
	"go.inet256.org/inet256/src/internal/peers"
	"go.inet256.org/inet256/src/internal/retry"
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

func ParseAddrs(ap AddrParser, xs []string) (ret []multiswarm.Addr, _ error) {
	for i := range xs {
		y, err := ap([]byte(xs[i]))
		if err != nil {
			return nil, err
		}
		ret = append(ret, y)
	}
	return ret, nil
}

func MarshalAddrs(xs []multiswarm.Addr) (ret []string) {
	for i := range xs {
		data, err := xs[i].MarshalText()
		if err != nil {
			panic(err)
		}
		ret = append(ret, string(data))
	}
	return ret
}
