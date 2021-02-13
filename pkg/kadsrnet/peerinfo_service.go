package kadsrnet

import (
	"context"
	sync "sync"

	"github.com/brendoncarroll/go-p2p"
)

type sendAlongFunc = func(ctx context.Context, dst Addr, r *Route, body *Body) error

type infoService struct {
	store      *PeerInfoStore
	routeTable RouteTable
	send       sendAlongFunc
	active     sync.Map
}

func newInfoService(store *PeerInfoStore, rt RouteTable, send sendAlongFunc) *infoService {
	return &infoService{
		routeTable: rt,
		store:      store,
		send:       send,
	}
}

func (s *infoService) get(ctx context.Context, target Addr, r *Route) (*PeerInfo, error) {
	if pinfo := s.store.Get(target); pinfo != nil {
		return pinfo, nil
	}
	// create future
	fut := newInfoFuture()
	s.putFut(target, fut)
	defer s.deleteFut(target)

	// send req
	if err := s.sendRequest(ctx, target, r); err != nil {
		return nil, err
	}
	return fut.get(ctx)
}

func (s *infoService) onPeerInfoReq(req *PeerInfoReq) *PeerInfo {
	return s.selfPeerInfo()
}

func (s *infoService) selfPeerInfo() *PeerInfo {
	return s.store.Get(s.routeTable.LocalAddr())
}

func (s *infoService) onPeerInfo(info *PeerInfo) error {
	addr, err := addrFromInfo(info)
	if err != nil {
		return err
	}
	s.store.Put(addr, info)
	fut := s.getFut(addr)
	if fut != nil {
		fut.complete(info)
	}
	return nil
}

func (s *infoService) sendRequest(ctx context.Context, target Addr, r *Route) error {
	if err := s.send(ctx, target, r, &Body{
		Body: &Body_PeerInfo{PeerInfo: s.selfPeerInfo()},
	}); err != nil {
		return err
	}
	return s.send(ctx, target, r, &Body{
		Body: &Body_PeerInfoReq{&PeerInfoReq{}},
	})
}

func (s *infoService) putFut(x Addr, fut *infoFuture) {
	s.active.Store(x, fut)
}

func (s *infoService) getFut(x Addr) *infoFuture {
	v, ok := s.active.Load(x)
	if !ok {
		return nil
	}
	return v.(*infoFuture)
}

func (s *infoService) deleteFut(x Addr) {
	s.active.Delete(x)
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
