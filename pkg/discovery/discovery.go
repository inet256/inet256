package discovery

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/stdctx/logctx"

	"github.com/inet256/inet256/internal/retry"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/peers"
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

type AddrDiscoveryParams struct {
	// Announcing
	PrivateKey    inet256.PrivateKey
	LocalID       inet256.Addr
	GetLocalAddrs func() []TransportAddr

	// Finding
	AddressBook AddressBook
	AddrParser  p2p.AddrParser[TransportAddr]
}

// AddrService find transport addresses for known peers
type AddrService interface {
	RunAddrDiscovery(ctx context.Context, params AddrDiscoveryParams) error
}

type PeerDiscoveryParams struct {
	// Outbound
	PrivateKey    inet256.PrivateKey
	LocalID       inet256.Addr
	GetLocalAddrs func() []TransportAddr

	// Inbound
	PeerStore peers.Store[TransportAddr]
	ParseAddr func([]byte) (TransportAddr, error)
}

// PeerService finds new peers and addresses
type PeerService interface {
	RunPeerDiscovery(ctx context.Context, params PeerDiscoveryParams) error
}

func RunForever(ctx context.Context, srv AddrService, params AddrDiscoveryParams) {
	logctx.Infoln(ctx, "starting address discovery service")
	retry.Retry(ctx, func() error {
		return srv.RunAddrDiscovery(ctx, params)
	}, retry.WithPredicate(func(err error) bool {
		logctx.Errorln(ctx, "error in address discovery service", err)
		return true
	}))
	logctx.Infoln(ctx, "stopped address discovery service")
}

func RunPeerDiscovery(ctx context.Context, srv PeerService, params PeerDiscoveryParams) {
	logctx.Infoln(ctx, "starting peer discovery service")
	retry.Retry(ctx, func() error {
		return srv.RunPeerDiscovery(ctx, params)
	}, retry.WithPredicate(func(err error) bool {
		logctx.Errorln(ctx, "error in discovery service", err)
		return true
	}))
	logctx.Infoln(ctx, "stopped peer discovery service")
}
