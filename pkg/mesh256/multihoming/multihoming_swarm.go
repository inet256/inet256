package multihoming

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/p2pmux"
	"github.com/brendoncarroll/go-tai64"
	"github.com/brendoncarroll/stdctx"
	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/pkg/errors"

	"github.com/inet256/inet256/internal/netutil"
	"github.com/inet256/inet256/pkg/inet256"
)

const (
	channelHeartbeat = 0
	channelData      = 1
)

type comparableAddr interface {
	p2p.Addr
	comparable
}

type PeerStore[K comparable, A p2p.Addr] interface {
	Contains(k K) bool
	ListPeers() []K
	ListAddrs(k K) []A
}

// Swarm manages a underlying swarm where many addresses A map to fewer Ks
// Swarm manages sending background heartbeats to monitor the status of peers at any of the As
// Swarm allows callers to send to the best A for a given K
type Swarm[A p2p.Addr, Pub any, K comparableAddr] struct {
	peerStore PeerStore[K, A]
	inner     p2p.SecureSwarm[A, Pub]
	groupBy   func(Pub) (K, error)

	localKey  K
	mux       p2pmux.SecureMux[A, uint16, Pub]
	lm        *linkMonitor[A, Pub, K]
	dataSwarm p2p.SecureSwarm[A, Pub]
	meters    meterSet[K]
}

type Params[A p2p.Addr, Pub any, K comparableAddr] struct {
	Background context.Context
	Inner      p2p.SecureSwarm[A, Pub]
	Peers      PeerStore[K, A]
	GroupBy    func(Pub) (K, error)
}

func New[A p2p.Addr, Pub any, K comparableAddr](params Params[A, Pub, K]) *Swarm[A, Pub, K] {
	localKey, err := params.GroupBy(params.Inner.PublicKey())
	if err != nil {
		panic(err)
	}
	mux := p2pmux.NewUint16SecureMux[A, Pub](params.Inner)
	return &Swarm[A, Pub, K]{
		peerStore: params.Peers,
		inner:     params.Inner,
		localKey:  localKey,
		groupBy:   params.GroupBy,

		mux:       mux,
		lm:        newLinkMonitor(stdctx.Child(params.Background, "linkMonitor"), mux.Open(channelHeartbeat), params.Peers, params.GroupBy),
		dataSwarm: mux.Open(channelData),
	}
}

func (s *Swarm[A, Pub, K]) Tell(ctx context.Context, dst K, v p2p.IOVec) error {
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

func (s *Swarm[A, Pub, K]) Receive(ctx context.Context, th func(p2p.Message[K])) error {
	for done := false; !done; {
		if err := s.dataSwarm.Receive(ctx, func(msg p2p.Message[A]) {
			srcPub, err := s.dataSwarm.LookupPublicKey(ctx, msg.Src)
			if err != nil {
				logctx.Warnln(ctx, "could not lookup public key, dropping message: ", err)
				return
			}
			srcKey, err := s.groupBy(srcPub)
			if err != nil {
				logctx.Warnln(ctx, err)
				return
			}
			if !s.peerStore.Contains(srcKey) {
				logctx.Warnf(ctx, "dropping message from unknown peer %v", srcKey)
				return
			}
			s.lm.Mark(srcKey, msg.Src, tai64.Now())
			s.meters.Rx(srcKey, len(msg.Payload))

			th(p2p.Message[K]{
				Src:     srcKey,
				Dst:     s.localKey,
				Payload: msg.Payload,
			})
			done = true
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Swarm[A, Pub, K]) PublicKey() Pub {
	return s.dataSwarm.PublicKey()
}

func (s *Swarm[A, Pub, K]) LocalAddr() K {
	return s.localKey
}

func (s *Swarm[A, Pub, K]) LocalAddrs() []K {
	return []K{s.LocalAddr()}
}

func (s *Swarm[A, Pub, K]) Close() error {
	var el netutil.ErrList
	el.Add(s.lm.Close())
	el.Add(s.dataSwarm.Close())
	el.Add(s.inner.Close())
	return el.Err()
}

func (s *Swarm[A, Pub, K]) MTU(ctx context.Context, target K) int {
	addr, err := s.selectAddr(ctx, target)
	if err != nil {
		// TODO: figure out what to do here
		return 512
	}
	return s.dataSwarm.MTU(ctx, *addr)
}

func (s *Swarm[A, Pub, K]) MaxIncomingSize() int {
	return s.inner.MaxIncomingSize()
}

func (s *Swarm[A, Pub, K]) LookupPublicKey(ctx context.Context, target K) (ret Pub, _ error) {
	if !s.peerStore.Contains(target) {
		return ret, p2p.ErrPublicKeyNotFound
	}
	addr, err := s.selectAddr(ctx, target)
	if err != nil {
		if inet256.IsErrUnreachable(err) {
			err = p2p.ErrPublicKeyNotFound
		}
		return ret, err
	}
	return s.dataSwarm.LookupPublicKey(ctx, *addr)
}

func (s *Swarm[A, Pub, K]) ParseAddr(data []byte) (K, error) {
	panic("not implemented")
}

func (s *Swarm[A, Pub, K]) selectAddr(ctx context.Context, k K) (*A, error) {
	return s.lm.PickAddr(ctx, k)
}

func (s *Swarm[A, Pub, K]) LastSeen(k K) map[string]tai64.TAI64N {
	return s.lm.LastSeen(k)
}

func (s *Swarm[A, Pub, K]) GetTx(k K) uint64 {
	return s.meters.Tx(k, 0)
}

func (s *Swarm[A, Pub, K]) GetRx(k K) uint64 {
	return s.meters.Rx(k, 0)
}
