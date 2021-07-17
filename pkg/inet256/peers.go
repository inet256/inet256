package inet256

import "github.com/brendoncarroll/go-p2p"

// PeerStore stores information about peers
type PeerStore interface {
	Add(x Addr)
	Remove(x Addr)
	SetAddrs(x Addr, addrs []p2p.Addr)
	ListAddrs(x Addr) []p2p.Addr

	PeerSet
}

// PeerSet represents a set of peers
type PeerSet interface {
	ListPeers() []Addr
	Contains(Addr) bool
}
