package peers

import (
	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

type Addr = inet256.Addr

// PeerStore stores information about peers
// TA is the type of the transport address
type Store [T p2p.Addr]interface {
	Add(x Addr)
	Remove(x Addr)
	SetAddrs(x Addr, addrs []T)
	ListAddrs(x Addr) []T

	Set
}

// PeerSet represents a set of peers
type Set interface {
	ListPeers() []Addr
	Contains(Addr) bool
}
