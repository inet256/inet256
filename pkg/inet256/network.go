package inet256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
)

type RecvFunc func(src, dst Addr, data []byte)

func NoOpRecvFunc(src, dst Addr, data []byte) {}

type PeerStore interface {
	ListPeers() []p2p.PeerID
	ListAddrs(p2p.PeerID) []string
}

type Network interface {
	SendTo(ctx context.Context, addr Addr, data []byte) error
	OnRecv(fn RecvFunc)
	AddrWithPrefix(ctx context.Context, prefix []byte, nbits int) (Addr, error)
	MinMTU() int
	Close() error
}

type NetworkParams struct {
	Swarm p2p.SecureSwarm
	Peers PeerStore
}

type NetworkFactory func(NetworkParams) Network

type NetworkSpec struct {
	Name    string
	Factory NetworkFactory
}
