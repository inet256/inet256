package inet256

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/intmux"
	"github.com/brendoncarroll/go-p2p/s/noiseswarm"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
)

// PeerSwarm is the type of a p2p.Swarm which uses p2p.PeerIDs as addresses
type PeerSwarm = peerswarm.Swarm

const (
	channelHeartbeat = 0
	channelData      = 127
)

var _ PeerSwarm = &peerSwarm{}

// peerSwarm manages the mapping from transport addresses to peerIDs.
// a peer can be reachable at many addresses in the transport swarm.
// Some of the addresses may not even be in the PeerStore
type peerSwarm struct {
	peerStore PeerStore
	inner     p2p.SecureSwarm

	localID   p2p.PeerID
	mux       intmux.SecureMux
	tm        *transportMonitor
	dataSwarm p2p.SecureSwarm
	meter     meter
}

func newPeerSwarm(x p2p.SecureSwarm, peerStore PeerStore) *peerSwarm {
	mux := intmux.WrapSecureSwarm(x)
	return &peerSwarm{
		peerStore: peerStore,
		inner:     x,
		localID:   p2p.NewPeerID(x.PublicKey()),

		mux:       mux,
		tm:        newTransportMonitor(mux.Open(channelHeartbeat), peerStore),
		dataSwarm: mux.Open(channelData),
	}
}

func (s *peerSwarm) Tell(ctx context.Context, dst p2p.Addr, v p2p.IOVec) error {
	return s.TellPeer(ctx, dst.(p2p.PeerID), v)
}

func (s *peerSwarm) TellPeer(ctx context.Context, dst p2p.PeerID, v p2p.IOVec) error {
	addr, err := s.selectAddr(dst)
	if err != nil {
		return err
	}
	s.meter.Tx(dst, p2p.VecSize(v))
	return s.dataSwarm.Tell(ctx, addr, v)
}

func (s *peerSwarm) ServeTells(fn p2p.TellHandler) error {
	return s.dataSwarm.ServeTells(func(msg *p2p.Message) {
		srcKey := p2p.LookupPublicKeyInHandler(s.dataSwarm, msg.Src)
		srcID := p2p.NewPeerID(srcKey)
		msg2 := &p2p.Message{
			Src:     srcID,
			Dst:     s.localID,
			Payload: msg.Payload,
		}
		s.tm.Mark(srcID, msg.Src, time.Now())
		s.meter.Rx(srcID, len(msg.Payload))
		fn(msg2)
	})
}

func (s *peerSwarm) PublicKey() p2p.PublicKey {
	return s.inner.PublicKey()
}

func (s *peerSwarm) LocalAddrs() []p2p.Addr {
	return []p2p.Addr{s.localID}
}

func (s *peerSwarm) ParseAddr(data []byte) (p2p.Addr, error) {
	id := p2p.PeerID{}
	if err := id.UnmarshalText(data); err != nil {
		return nil, err
	}
	return id, nil
}

func (s *peerSwarm) Close() error {
	return s.inner.Close()
}

func (s *peerSwarm) MTU(ctx context.Context, target p2p.Addr) int {
	addr, err := s.selectAddr(target.(p2p.PeerID))
	if err != nil {
		// TODO: figure out what to do here
		return 512
	}
	return s.dataSwarm.MTU(ctx, addr)
}

func (s *peerSwarm) LookupPublicKey(ctx context.Context, target p2p.Addr) (p2p.PublicKey, error) {
	if !s.peerStore.Contains(target.(p2p.PeerID)) {
		return nil, p2p.ErrPublicKeyNotFound
	}
	addr, err := s.selectAddr(target.(p2p.PeerID))
	if err != nil {
		return nil, err
	}
	return s.dataSwarm.LookupPublicKey(ctx, addr)
}

func (s *peerSwarm) selectAddr(x p2p.PeerID) (p2p.Addr, error) {
	return s.tm.PickAddr(x)
}

func (s *peerSwarm) LastSeen(id p2p.PeerID) map[string]time.Time {
	return s.tm.LastSeen(id)
}

func (s *peerSwarm) GetTx(id p2p.PeerID) uint64 {
	return s.meter.Tx(id, 0)
}

func (s *peerSwarm) GetRx(id p2p.PeerID) uint64 {
	return s.meter.Rx(id, 0)
}

type peerSwarmWrapper struct {
	p2p.SecureSwarm
}

func castPeerSwarm(x p2p.SecureSwarm) peerSwarmWrapper {
	return peerSwarmWrapper{x}
}

func (s peerSwarmWrapper) TellPeer(ctx context.Context, id p2p.PeerID, v p2p.IOVec) error {
	return s.SecureSwarm.Tell(ctx, id, v)
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

func (s noise2PeerSwarm) ServeTells(fn p2p.TellHandler) error {
	return s.Swarm.ServeTells(func(x *p2p.Message) {
		fn(&p2p.Message{
			Src:     x.Src.(noiseswarm.Addr).ID,
			Dst:     x.Dst.(noiseswarm.Addr).ID,
			Payload: x.Payload,
		})
	})
}

func (s noise2PeerSwarm) TellPeer(ctx context.Context, id p2p.PeerID, data p2p.IOVec) error {
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
