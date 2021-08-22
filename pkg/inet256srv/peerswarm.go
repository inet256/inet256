package inet256srv

import (
	"bytes"
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/p2pmux"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	channelHeartbeat = 0
	channelData      = 1
)

// swarm manages the mapping from transport addresses p2p.Addr to inet256.Addrs.
// a peer can be reachable at many addresses in the transport swarm.
// Some of the addresses may not even be in the PeerStore
type swarm struct {
	peerStore PeerStore
	inner     p2p.SecureSwarm

	localID   inet256.Addr
	mux       p2pmux.Uint16SecureMux
	lm        *linkMonitor
	dataSwarm p2p.SecureSwarm
	meters    meterSet
}

func NewSwarm(x p2p.SecureSwarm, peerStore PeerStore) inet256.Swarm {
	return swarmWrapper{newSwarm(x, peerStore)}
}

func newSwarm(x p2p.SecureSwarm, peerStore PeerStore) *swarm {
	mux := p2pmux.NewUint16SecureMux(x)
	return &swarm{
		peerStore: peerStore,
		inner:     x,
		localID:   inet256.NewAddr(x.PublicKey()),

		mux:       mux,
		lm:        newLinkMonitor(mux.Open(channelHeartbeat), peerStore, logrus.StandardLogger()),
		dataSwarm: mux.Open(channelData),
	}
}

func (s *swarm) Tell(ctx context.Context, dst p2p.Addr, v p2p.IOVec) error {
	dst2 := dst.(inet256.Addr)
	if !s.peerStore.Contains(dst2) {
		return errors.Errorf("tell to unrecognized peer %v", dst2)
	}
	addr, err := s.selectAddr(dst2)
	if err != nil {
		return err
	}
	s.meters.Tx(dst2, p2p.VecSize(v))
	return s.dataSwarm.Tell(ctx, addr, v)
}

func (s *swarm) Receive(ctx context.Context, src, dst *p2p.Addr, buf []byte) (int, error) {
	for {
		n, err := s.dataSwarm.Receive(ctx, src, dst, buf)
		if err != nil {
			return 0, err
		}
		srcKey, err := s.dataSwarm.LookupPublicKey(ctx, *src)
		if err != nil {
			logrus.Error("could not lookup public key, dropping message: ", err)
			continue
		}
		srcID := inet256.NewAddr(srcKey)
		if !s.peerStore.Contains(srcID) {
			logrus.Warnf("dropping message from unknown peer %v", srcID)
			continue
		}
		s.lm.Mark(srcID, *src, time.Now())
		s.meters.Rx(inet256.Addr(srcID), n)

		*src = srcID
		*dst = s.localID
		return n, nil
	}
}

func (s *swarm) PublicKey() p2p.PublicKey {
	return s.inner.PublicKey()
}

func (s *swarm) LocalAddrs() []p2p.Addr {
	return []p2p.Addr{s.localID}
}

func (s *swarm) ParseAddr(data []byte) (p2p.Addr, error) {
	id := inet256.Addr{}
	if err := id.UnmarshalText(data); err != nil {
		return nil, err
	}
	return id, nil
}

func (s *swarm) Close() error {
	var el netutil.ErrList
	el.Do(s.lm.Close)
	el.Do(s.dataSwarm.Close)
	el.Do(s.inner.Close)
	return el.Err()
}

func (s *swarm) MTU(ctx context.Context, target p2p.Addr) int {
	addr, err := s.selectAddr(target.(inet256.Addr))
	if err != nil {
		// TODO: figure out what to do here
		return 512
	}
	return s.dataSwarm.MTU(ctx, addr)
}

func (s *swarm) MaxIncomingSize() int {
	return s.inner.MaxIncomingSize()
}

func (s *swarm) LookupPublicKey(ctx context.Context, target p2p.Addr) (p2p.PublicKey, error) {
	if !s.peerStore.Contains(target.(inet256.Addr)) {
		return nil, p2p.ErrPublicKeyNotFound
	}
	addr, err := s.selectAddr(target.(inet256.Addr))
	if err != nil {
		return nil, err
	}
	return s.dataSwarm.LookupPublicKey(ctx, addr)
}

func (s *swarm) selectAddr(x inet256.Addr) (p2p.Addr, error) {
	return s.lm.PickAddr(x)
}

func (s *swarm) LastSeen(id inet256.Addr) map[string]time.Time {
	return s.lm.LastSeen(id)
}

func (s *swarm) GetTx(id inet256.Addr) uint64 {
	return s.meters.Tx(id, 0)
}

func (s *swarm) GetRx(id inet256.Addr) uint64 {
	return s.meters.Rx(id, 0)
}

// swarmWrapper converts a p2p.SecureSwarm to an inet256.Swarm
type swarmWrapper struct {
	s p2p.SecureSwarm
}

func (s swarmWrapper) Tell(ctx context.Context, dst Addr, m p2p.IOVec) error {
	return s.s.Tell(ctx, dst, m)
}

func (s swarmWrapper) Receive(ctx context.Context, src, dst *Addr, buf []byte) (int, error) {
	var src2, dst2 p2p.Addr
	n, err := s.s.Receive(ctx, &src2, &dst2, buf)
	if err != nil {
		return n, err
	}
	*src = src2.(Addr)
	*dst = dst2.(Addr)
	return n, err
}

func (s swarmWrapper) LookupPublicKey(ctx context.Context, target Addr) (inet256.PublicKey, error) {
	return s.s.LookupPublicKey(ctx, target)
}

func (s swarmWrapper) LocalAddr() Addr {
	return s.s.LocalAddrs()[0].(Addr)
}

func (s swarmWrapper) Close() error {
	return s.s.Close()
}

func newSecureNetwork(privateKey inet256.PrivateKey, x Network) Network {
	insecSwarm := SwarmFromNetwork(x, privateKey.Public())
	quicSw, err := quicswarm.New(insecSwarm, privateKey)
	if err != nil {
		panic(err)
	}
	secnet := networkFromSwarm(quic2Swarm{quicSw}, x.FindAddr, x.Bootstrap)
	return secnet
}

// quic2Swarm turns quicswarm.Addr into inet256.Addr
type quic2Swarm struct {
	*quicswarm.Swarm
}

func (s quic2Swarm) Receive(ctx context.Context, src, dst *p2p.Addr, buf []byte) (int, error) {
	for {
		n, err := s.Swarm.Receive(ctx, src, dst, buf)
		if err != nil {
			return 0, err
		}
		// This is where the actual check for who can send as what address happens
		srcID := p2p.ExtractPeerID(*src)
		srcAddr := (*src).(quicswarm.Addr).Addr.(inet256.Addr)
		if !bytes.Equal(srcID[:], srcAddr[:]) {
			logrus.Warnf("incorrect id=%v for address=%v", srcID, srcAddr)
			continue
		}
		*src = srcAddr
		*dst = (*dst).(quicswarm.Addr).Addr
		return n, nil
	}
}

func (s quic2Swarm) Tell(ctx context.Context, x p2p.Addr, data p2p.IOVec) error {
	dst := s.makeAddr(x.(inet256.Addr))
	return s.Swarm.Tell(ctx, dst, data)
}

func (s quic2Swarm) LocalAddr() p2p.Addr {
	return s.Swarm.LocalAddrs()[0].(quicswarm.Addr).Addr
}

func (s quic2Swarm) LocalAddrs() (ys []p2p.Addr) {
	for _, x := range s.Swarm.LocalAddrs() {
		ys = append(ys, x.(quicswarm.Addr).Addr)
	}
	return ys
}

func (s quic2Swarm) LookupPublicKey(ctx context.Context, target p2p.Addr) (p2p.PublicKey, error) {
	x := s.makeAddr(target.(inet256.Addr))
	return s.Swarm.LookupPublicKey(ctx, x)
}

func (s quic2Swarm) ParseAddr(x []byte) (p2p.Addr, error) {
	id := inet256.Addr{}
	if err := id.UnmarshalText(x); err != nil {
		return nil, err
	}
	return id, nil
}

func (s quic2Swarm) makeAddr(addr inet256.Addr) quicswarm.Addr {
	id := p2p.PeerID(addr)
	return quicswarm.Addr{
		ID:   id,
		Addr: addr,
	}
}
