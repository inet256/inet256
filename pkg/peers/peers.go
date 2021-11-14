package peers

import (
	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

type Addr = inet256.Addr

// PeerStore stores information about peers
type Store interface {
	Add(x Addr)
	Remove(x Addr)
	SetAddrs(x Addr, addrs []p2p.Addr)
	ListAddrs(x Addr) []p2p.Addr

	Set
}

// PeerSet represents a set of peers
type Set interface {
	ListPeers() []Addr
	Contains(Addr) bool
}
