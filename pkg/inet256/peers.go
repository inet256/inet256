package inet256

import "github.com/brendoncarroll/go-p2p"

// PeerStore stores information about peers
type PeerStore interface {
	Add(id p2p.PeerID)
	Remove(id p2p.PeerID)
	SetAddrs(id p2p.PeerID, addrs []string)
	ListAddrs(p2p.PeerID) []string

	PeerSet
}

// PeerSet represents a set of peers
type PeerSet interface {
	ListPeers() []p2p.PeerID
	Contains(p2p.PeerID) bool
}
