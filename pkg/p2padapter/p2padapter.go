// package p2padapter provides adapters for github.com/brendoncarroll/go-p2p
package p2padapter

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

// SwarmFromNode converts an inet256.Node to a p2p.SecureSwarm[inet256.Addr]
func SwarmFromNode(node inet256.Node) p2p.SecureSwarm[inet256.Addr] {
	return p2pNode{Node: node}
}

type p2pNode struct {
	inet256.Node
	extraSwarmMethods
}

func (pn p2pNode) Tell(ctx context.Context, dst inet256.Addr, v p2p.IOVec) error {
	return pn.Node.Send(ctx, dst, p2p.VecBytes(nil, v))
}

func (pn p2pNode) Receive(ctx context.Context, fn func(p2p.Message[inet256.Addr])) error {
	return pn.Node.Receive(ctx, func(msg inet256.Message) {
		fn(p2p.Message[inet256.Addr]{
			Src:     msg.Src,
			Dst:     msg.Dst,
			Payload: msg.Payload,
		})
	})
}

func (pn p2pNode) LocalAddrs() []inet256.Addr {
	return []inet256.Addr{pn.Node.LocalAddr()}
}

type extraSwarmMethods struct{}

func (extraSwarmMethods) MaxIncomingSize() int {
	return inet256.MaxMTU
}

func (extraSwarmMethods) ParseAddr(data []byte) (inet256.Addr, error) {
	var addr inet256.Addr
	if err := addr.UnmarshalText(data); err != nil {
		return inet256.Addr{}, err
	}
	return addr, nil
}
