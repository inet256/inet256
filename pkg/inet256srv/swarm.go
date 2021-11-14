package inet256srv

import (
	"bytes"
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/p2pmux"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/inet256/inet256/networks"
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

func NewSwarm(x p2p.SecureSwarm, peerStore PeerStore) networks.Swarm {
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
	err = s.dataSwarm.Tell(ctx, addr, v)
	if errors.Is(err, context.DeadlineExceeded) {
		err = nil
	}
	return err
}

func (s *swarm) Receive(ctx context.Context, th p2p.TellHandler) error {
	for done := false; !done; {
		if err := s.dataSwarm.Receive(ctx, func(msg p2p.Message) {
			srcKey, err := s.dataSwarm.LookupPublicKey(ctx, msg.Src)
			if err != nil {
				logrus.Error("could not lookup public key, dropping message: ", err)
				return
			}
			srcID := inet256.NewAddr(srcKey)
			if !s.peerStore.Contains(srcID) {
				logrus.Warnf("dropping message from unknown peer %v", srcID)
				return
			}
			s.lm.Mark(srcID, msg.Src, time.Now())
			s.meters.Rx(inet256.Addr(srcID), len(msg.Payload))

			th(p2p.Message{
				Src:     srcID,
				Dst:     s.localID,
				Payload: msg.Payload,
			})
			done = true
		}); err != nil {
			return err
		}
	}
	return nil
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
		if inet256.IsErrUnreachable(err) {
			err = inet256.ErrPublicKeyNotFound
		}
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

func (s swarmWrapper) Receive(ctx context.Context, th inet256.ReceiveFunc) error {
	return s.s.Receive(ctx, func(m p2p.Message) {
		th(inet256.Message{
			Src:     m.Src.(inet256.Addr),
			Dst:     m.Dst.(inet256.Addr),
			Payload: m.Payload,
		})
	})
}

func (s swarmWrapper) LookupPublicKey(ctx context.Context, target Addr) (inet256.PublicKey, error) {
	return s.s.LookupPublicKey(ctx, target)
}

func (s swarmWrapper) PublicKey() inet256.PublicKey {
	return s.s.PublicKey()
}

func (s swarmWrapper) LocalAddr() Addr {
	return s.s.LocalAddrs()[0].(Addr)
}

func (s swarmWrapper) Close() error {
	return s.s.Close()
}

func newSecureNetwork(privateKey inet256.PrivateKey, x Network) Network {
	insecSwarm := SwarmFromNode(x)
	fingerprinter := func(pubKey inet256.PublicKey) p2p.PeerID {
		return p2p.PeerID(inet256.NewAddr(pubKey))
	}
	quicSw, err := quicswarm.New(insecSwarm, privateKey, quicswarm.WithFingerprinter(fingerprinter))
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

func (s quic2Swarm) Receive(ctx context.Context, th p2p.TellHandler) error {
	for done := false; !done; {
		if err := s.Swarm.Receive(ctx, func(msg p2p.Message) {
			srcID := p2p.ExtractPeerID(msg.Src)
			srcAddr := (msg.Src).(quicswarm.Addr).Addr.(inet256.Addr)
			// This is where the actual check for who can send as what address happens
			if !bytes.Equal(srcID[:], srcAddr[:]) {
				logrus.Warnf("incorrect id=%v for address=%v", srcID, srcAddr)
				return
			}
			th(p2p.Message{
				Src:     srcAddr,
				Dst:     msg.Dst.(quicswarm.Addr).Addr.(inet256.Addr),
				Payload: msg.Payload,
			})
			done = true
		}); err != nil {
			return err
		}
	}
	return nil
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
