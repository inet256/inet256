package mesh256

import (
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/peers"
)

type (
	Addr          = inet256.Addr
	Node          = inet256.Node
	TransportAddr = multiswarm.Addr

	Network       = networks.Network
	NetworkParams = networks.Params
	PeerSet       = networks.PeerSet
)

func NewPeerStore() peers.Store[TransportAddr] {
	return peers.NewStore[TransportAddr]()
}
