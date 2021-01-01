package kadsrnet

import (
	"context"
	sync "sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"
)

type sendLookupPeerFunc = func(ctx context.Context, dst Addr, lp *LookupPeerReq) error

type infoService struct {
	store      *PeerInfoStore
	routeTable RouteTable
	send       sendLookupPeerFunc
	active     sync.Map
}

func newInfoService(store *PeerInfoStore, rt RouteTable, send sendLookupPeerFunc) *infoService {
	return &infoService{
		routeTable: rt,
		store:      store,
		send:       send,
	}
}

func (s *infoService) lookup(ctx context.Context, prefix []byte, nbits int, hint *Addr) (*PeerInfo, error) {
	if pinfo := s.store.Lookup(prefix, nbits); pinfo != nil {
		return pinfo, nil
	}
	// create future
	fut := newInfoFuture()
	s.put(prefix, fut)
	defer s.delete(prefix)
	// do the lookup
	r := s.routeTable.RouteTo(idFromBytes(prefix))
	if r == nil && hint != nil {
		r = s.routeTable.RouteTo(*hint)
	}
	if r == nil {
		return nil, errors.Errorf("cannot lookup info, no closer peer")
	}
	s.send(ctx, idFromBytes(r.Dst), &LookupPeerReq{
		Prefix: prefix,
		Nbits:  uint32(nbits),
	})
	// await future result
	return fut.get(ctx)
}

func (s *infoService) onLookupPeer(req *LookupPeerReq) *LookupPeerRes {
	pinfo := s.store.Lookup(req.Prefix, int(req.Nbits))
	if pinfo == nil {
		return nil
	}
	return &LookupPeerRes{
		Prefix:   req.Prefix,
		Nbits:    req.Nbits,
		PeerInfo: pinfo,
	}
}

func (s *infoService) onLookupPeerRes(res *LookupPeerRes) error {
	addr, err := addrFromInfo(res.PeerInfo)
	if err != nil {
		return err
	}
	s.store.Put(addr, res.PeerInfo)
	fut := s.get(res.Prefix)
	if fut != nil {
		fut.complete(res.PeerInfo)
	}
	return nil
}

func (s *infoService) put(x []byte, fut *infoFuture) {
	s.active.Store(string(x[:]), fut)
}

func (s *infoService) get(x []byte) *infoFuture {
	v, ok := s.active.Load(string(x))
	if !ok {
		return nil
	}
	return v.(*infoFuture)
}

func (s *infoService) delete(x []byte) {
	s.active.Delete(string(x))
}

type infoFuture struct {
	done   chan struct{}
	result *PeerInfo
	once   sync.Once
}

func newInfoFuture() *infoFuture {
	return &infoFuture{
		done: make(chan struct{}),
	}
}

func (f *infoFuture) wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-f.done:
		return nil
	}
}

func (f *infoFuture) get(ctx context.Context) (*PeerInfo, error) {
	if err := f.wait(ctx); err != nil {
		return nil, err
	}
	return f.result, nil
}

func (f *infoFuture) complete(res *PeerInfo) {
	f.once.Do(func() {
		f.result = res
		close(f.done)
	})
}

func addrFromInfo(info *PeerInfo) (Addr, error) {
	pubKey, err := p2p.ParsePublicKey(info.PublicKey)
	if err != nil {
		return Addr{}, nil
	}
	return p2p.NewPeerID(pubKey), nil
}
