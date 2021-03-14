package kadsrnet

import (
	"context"
	sync "sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
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

func (s *infoService) findAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	if kademlia.HasPrefix(s.store.localAddr[:], prefix, nbits) {
		return s.store.localAddr, nil
	}
	for _, peerID := range s.store.oneHopPeers.ListPeers() {
		if kademlia.HasPrefix(peerID[:], prefix, nbits) {
			return hashPublicKey(s.store.Get(peerID).PublicKey), nil
		}
	}
	if peerInfo := s.store.Lookup(prefix, nbits); peerInfo != nil {
		return hashPublicKey(peerInfo.PublicKey), nil
	}
	return Addr{}, inet256.ErrNoAddrWithPrefix
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

func (s *infoService) onPeerInfoReq(req *PeerInfoReq) (*PeerInfo, error) {
	pubKey, err := inet256.ParsePublicKey(req.GetAskerPublicKey())
	if err != nil {
		return nil, errors.Errorf("info requests must contain valid public key")
	}
	s.store.Put(inet256.NewAddr(pubKey), &PeerInfo{PublicKey: req.AskerPublicKey})
	return s.selfPeerInfo(), nil
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
	return s.send(ctx, target, r, &Body{
		Body: &Body_PeerInfoReq{&PeerInfoReq{
			AskerPublicKey: s.selfPeerInfo().PublicKey,
		}},
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
	future
}

func newInfoFuture() *infoFuture {
	return &infoFuture{*newFuture()}
}

func (f *infoFuture) get(ctx context.Context) (*PeerInfo, error) {
	x, err := f.future.get(ctx)
	if err != nil {
		return nil, err
	}
	return x.(*PeerInfo), nil
}

func (f *infoFuture) complete(res *PeerInfo) {
	f.future.complete(res)
}

func addrFromInfo(info *PeerInfo) (Addr, error) {
	pubKey, err := p2p.ParsePublicKey(info.PublicKey)
	if err != nil {
		return Addr{}, nil
	}
	return p2p.NewPeerID(pubKey), nil
}

func hashPublicKey(x []byte) Addr {
	y := Addr{}
	sha3.ShakeSum256(y[:], x)
	return y
}
