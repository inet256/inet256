package mesh256

import (
	"context"
	"os"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/p2pmux"
	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/inet256/inet256/pkg/peers"
)

const (
	channelHeartbeat = 0
	channelData      = 1
)

// swarm implements Swarm, which is the interface passed to networks.
//
// swarm manages the mapping from transport addresses T to inet256.Addrs.
// a peer can be reachable at many addresses in the transport swarm.
// Some of the addresses may not even be in the PeerStore
type swarm[T p2p.Addr] struct {
	peerStore peers.Store[T]
	inner     p2p.SecureSwarm[T]

	localID   inet256.Addr
	mux       p2pmux.SecureMux[T, uint16]
	lm        *linkMonitor[T]
	dataSwarm p2p.SecureSwarm[T]
	meters    meterSet
	extraSwarmMethods
}

func newSwarm[T p2p.Addr](x p2p.SecureSwarm[T], peerStore peers.Store[T]) *swarm[T] {
	pubKey, err := inet256.PublicKeyFromBuiltIn(x.PublicKey())
	if err != nil {
		panic(err)
	}
	mux := p2pmux.NewUint16SecureMux[T](x)
	return &swarm[T]{
		peerStore: peerStore,
		inner:     x,
		localID:   inet256.NewAddr(pubKey),

		mux:       mux,
		lm:        newLinkMonitor(mux.Open(channelHeartbeat), peerStore, slog.New(slog.NewTextHandler(os.Stderr))),
		dataSwarm: mux.Open(channelData),
	}
}

func (s *swarm[T]) Tell(ctx context.Context, dst inet256.Addr, v p2p.IOVec) error {
	if !s.peerStore.Contains(dst) {
		return errors.Errorf("tell to unrecognized peer %v", dst)
	}
	addr, err := s.selectAddr(ctx, dst)
	if err != nil {
		return err
	}
	s.meters.Tx(dst, p2p.VecSize(v))
	err = s.dataSwarm.Tell(ctx, *addr, v)
	if errors.Is(err, context.DeadlineExceeded) {
		err = nil
	}
	return err
}

func (s *swarm[T]) Receive(ctx context.Context, th func(p2p.Message[inet256.Addr])) error {
	for done := false; !done; {
		if err := s.dataSwarm.Receive(ctx, func(msg p2p.Message[T]) {
			srcKey, err := s.dataSwarm.LookupPublicKey(ctx, msg.Src)
			if err != nil {
				logctx.Warnln(ctx, "could not lookup public key, dropping message: ", err)
				return
			}
			srcKey2, err := inet256.PublicKeyFromBuiltIn(srcKey)
			if err != nil {
				logctx.Warnln(ctx, err)
				return
			}
			srcID := inet256.NewAddr(srcKey2)
			if !s.peerStore.Contains(srcID) {
				logctx.Warnf(ctx, "dropping message from unknown peer %v", srcID)
				return
			}
			s.lm.Mark(srcID, msg.Src, time.Now())
			s.meters.Rx(inet256.Addr(srcID), len(msg.Payload))

			th(p2p.Message[inet256.Addr]{
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

func (s *swarm[T]) PublicKey() p2p.PublicKey {
	return s.inner.PublicKey()
}

func (s *swarm[T]) LocalAddr() inet256.Addr {
	return s.localID
}

func (s *swarm[T]) LocalAddrs() []inet256.Addr {
	return []inet256.Addr{s.LocalAddr()}
}

func (s *swarm[T]) Close() error {
	var el netutil.ErrList
	el.Add(s.lm.Close())
	el.Add(s.dataSwarm.Close())
	el.Add(s.inner.Close())
	return el.Err()
}

func (s *swarm[T]) MTU(ctx context.Context, target inet256.Addr) int {
	addr, err := s.selectAddr(ctx, target)
	if err != nil {
		// TODO: figure out what to do here
		return 512
	}
	return s.dataSwarm.MTU(ctx, *addr)
}

func (s *swarm[T]) MaxIncomingSize() int {
	return s.inner.MaxIncomingSize()
}

func (s *swarm[T]) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	if !s.peerStore.Contains(target) {
		return nil, p2p.ErrPublicKeyNotFound
	}
	addr, err := s.selectAddr(ctx, target)
	if err != nil {
		if inet256.IsErrUnreachable(err) {
			err = inet256.ErrPublicKeyNotFound
		}
		return nil, err
	}
	return s.dataSwarm.LookupPublicKey(ctx, *addr)
}

func (s *swarm[T]) selectAddr(ctx context.Context, x inet256.Addr) (*T, error) {
	return s.lm.PickAddr(ctx, x)
}

func (s *swarm[T]) LastSeen(id inet256.Addr) map[string]time.Time {
	return s.lm.LastSeen(id)
}

func (s *swarm[T]) GetTx(id inet256.Addr) uint64 {
	return s.meters.Tx(id, 0)
}

func (s *swarm[T]) GetRx(id inet256.Addr) uint64 {
	return s.meters.Rx(id, 0)
}
