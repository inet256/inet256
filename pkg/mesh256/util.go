package mesh256

import (
	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/peers"
)

func NewSwarm[T p2p.Addr](x p2p.SecureSwarm[T], peers peers.Store[T]) Swarm {
	sw := newSwarm(x, peers)
	return swarmFromP2P(sw)
}
