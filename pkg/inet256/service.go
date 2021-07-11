package inet256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
)

type Node interface {
	Network

	ListOneHop() []p2p.PeerID
}

type Service interface {
	CreateNode(ctx context.Context, privKey p2p.PrivateKey) (Node, error)
	DeleteNode(privKey p2p.PrivateKey) error

	LookupPublicKey(ctx context.Context, addr Addr) (p2p.PublicKey, error)
	FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error)
	MTU(ctx context.Context, addr Addr) int
}
