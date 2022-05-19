// package mesh256 implements an INET256 Service in terms of a distributed routing algorithm
// and a dynamic set of one hop connections to peers.
package mesh256

import (
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/peers"
)

type (
	Addr          = inet256.Addr
	Node          = inet256.Node
	TransportAddr = multiswarm.Addr

	PeerSet = peers.Set

	PublicKey  = inet256.PublicKey
	PrivateKey = inet256.PrivateKey
)

const (
	TransportMTU = (1 << 16) - 1

	MinMTU = inet256.MinMTU
	MaxMTU = inet256.MaxMTU
)

func NewPeerStore() peers.Store[TransportAddr] {
	return peers.NewStore[TransportAddr]()
}