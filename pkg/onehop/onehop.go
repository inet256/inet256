package onehop

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var _ inet256.Network = &Network{}

func Factory(params inet256.NetworkParams) inet256.Network {
	return New(params.Swarm, params.Peers)
}

type Network struct {
	swarm     p2p.SecureSwarm
	peers     inet256.PeerStore
	peerSwarm *peerswarm.Swarm

	onRecv inet256.RecvFunc
}

func New(sw p2p.SecureSwarm, peers inet256.PeerStore) *Network {
	n := &Network{
		swarm: sw,
		peers: peers,
		peerSwarm: peerswarm.NewSwarm(sw, func(id p2p.PeerID) (addrs []p2p.Addr) {
			for _, addrStr := range peers.ListAddrs(id) {
				addr, err := sw.ParseAddr([]byte(addrStr))
				if err != nil {
					log.Error(err)
					continue
				}
				addrs = append(addrs, addr)
			}
			return addrs
		}),
		onRecv: inet256.NoOpRecvFunc,
	}
	n.peerSwarm.OnTell(func(msg *p2p.Message) {
		dst := msg.Dst.(p2p.PeerID)
		src := msg.Src.(p2p.PeerID)
		n.onRecv(src, dst, msg.Payload)
	})
	return n
}

func (n *Network) AddrWithPrefix(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	for _, id := range n.peers.ListPeers() {
		addr := id
		if inet256.HasPrefix(addr, prefix, nbits) {
			return addr, nil
		}
	}
	return inet256.Addr{}, inet256.ErrAddrUnreachable
}

func (n *Network) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	for _, id := range n.peers.ListPeers() {
		if id == target {
			pubKey := n.peerSwarm.LookupPublicKey(id)
			if pubKey == nil {
				return nil, errors.Errorf("unable to lookup public key")
			}
		}
	}
	return nil, inet256.ErrAddrUnreachable
}

func (n *Network) Tell(ctx context.Context, dst inet256.Addr, data []byte) error {
	return n.peerSwarm.TellPeer(ctx, dst, data)
}

func (n *Network) OnRecv(fn inet256.RecvFunc) {
	n.onRecv = fn
}

func (n *Network) Close() error {
	return n.peerSwarm.Close()
}
