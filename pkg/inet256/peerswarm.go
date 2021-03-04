package inet256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/noiseswarm"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
)

// PeerSwarm is the type of a p2p.Swarm which uses p2p.PeerIDs as addresses
type PeerSwarm = peerswarm.Swarm

func NewPeerSwarm(x p2p.SecureSwarm, addrSource peerswarm.AddrSource) PeerSwarm {
	return peerswarm.NewSwarm(x, addrSource)
}

type noise2PeerSwarm struct {
	*noiseswarm.Swarm
}

func newSecureNetwork(privateKey p2p.PrivateKey, x Network) Network {
	insecSw := SwarmFromNetwork(x, privateKey.Public())
	noiseSw := noiseswarm.New(insecSw, privateKey)
	secnet := networkFromSwarm(noise2PeerSwarm{noiseSw}, x.FindAddr, x.WaitReady)
	return secnet
}

func (s noise2PeerSwarm) OnTell(fn p2p.TellHandler) {
	s.Swarm.OnTell(func(x *p2p.Message) {
		fn(&p2p.Message{
			Src:     x.Src.(noiseswarm.Addr).ID,
			Dst:     x.Dst.(noiseswarm.Addr).ID,
			Payload: x.Payload,
		})
	})
}

func (s noise2PeerSwarm) TellPeer(ctx context.Context, id p2p.PeerID, data []byte) error {
	return s.Swarm.Tell(ctx, noiseswarm.Addr{ID: id, Addr: id}, data)
}

func (s noise2PeerSwarm) LocalAddrs() (ys []p2p.Addr) {
	for _, x := range s.Swarm.LocalAddrs() {
		ys = append(ys, x.(noiseswarm.Addr).ID)
	}
	return ys
}

func (s noise2PeerSwarm) LookupPublicKey(ctx context.Context, target p2p.Addr) (p2p.PublicKey, error) {
	id := target.(p2p.PeerID)
	return s.Swarm.LookupPublicKey(ctx, noiseswarm.Addr{ID: id, Addr: id})
}

func (s noise2PeerSwarm) LocalID() p2p.PeerID {
	return s.Swarm.LocalAddrs()[0].(noiseswarm.Addr).ID
}

func (s noise2PeerSwarm) ParseAddr(x []byte) (p2p.Addr, error) {
	id := p2p.PeerID{}
	if err := id.UnmarshalText(x); err != nil {
		return nil, err
	}
	return id, nil
}
