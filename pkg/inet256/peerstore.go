package inet256

import (
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/sirupsen/logrus"
)

// PeerStore stores information about peers
type PeerStore interface {
	Add(id p2p.PeerID)
	Remove(id p2p.PeerID)
	SetAddrs(id p2p.PeerID, addrs []string)
	ListAddrs(p2p.PeerID) []string

	PeerSet
}

// PeerSet represents a set of peers
type PeerSet interface {
	ListPeers() []p2p.PeerID
	Contains(p2p.PeerID) bool
}

type peerStore struct {
	mu sync.RWMutex
	m  map[p2p.PeerID][]string
}

func NewPeerStore() PeerStore {
	return &peerStore{
		m: map[p2p.PeerID][]string{},
	}
}

func (s *peerStore) Add(id p2p.PeerID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.m[id]; !exists {
		s.m[id] = []string{}
	}
}

func (s *peerStore) Remove(id p2p.PeerID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.m[id]; !exists {
		s.m[id] = []string{}
	}
}

func (s *peerStore) AddAddr(id p2p.PeerID, addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	addrs := s.m[id]
	for i := range addrs {
		if addr == addrs[i] {
			return
		}
	}
	s.m[id] = append(s.m[id], addr)
}

func (s *peerStore) SetAddrs(id p2p.PeerID, addrs []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = addrs
}

func (s *peerStore) ListPeers() []p2p.PeerID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := []p2p.PeerID{}
	for id := range s.m {
		ids = append(ids, id)
	}
	return ids
}

func (s *peerStore) Contains(id p2p.PeerID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.m[id]
	return exists
}

func (s *peerStore) ListAddrs(id p2p.PeerID) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[id]
}

func NewAddrSource(swarm p2p.Swarm, store PeerStore) peerswarm.AddrSource {
	return func(id p2p.PeerID) []p2p.Addr {
		xs := store.ListAddrs(id)
		var ys []p2p.Addr
		for i := range xs {
			y, err := swarm.ParseAddr([]byte(xs[i]))
			if err != nil {
				logrus.Infof("%+v", swarm)
				logrus.Errorf("parsing addr %v, got error %v", xs[i], err)
				panic(xs[i])
				continue
			}
			ys = append(ys, y)
		}
		return ys
	}
}

var _ PeerStore = ChainPeerStore{}

type ChainPeerStore []PeerStore

func (ps ChainPeerStore) Add(p2p.PeerID) {
	panic("cannot Add to ChainPeerStore")
}

func (ps ChainPeerStore) Remove(p2p.PeerID) {
	panic("cannot Remove from ChainPeerStore")
}

func (ps ChainPeerStore) SetAddrs(id p2p.PeerID, addrs []string) {
	panic("cannot SetAddrs on ChainPeerStore")
}

func (ps ChainPeerStore) ListPeers() (ids []p2p.PeerID) {
	m := map[p2p.PeerID]struct{}{}
	for _, ps2 := range ps {
		for _, id := range ps2.ListPeers() {
			m[id] = struct{}{}
		}
	}
	ret := make([]p2p.PeerID, 0, len(m))
	for id := range m {
		ret = append(ret, id)
	}
	return ids
}

func (ps ChainPeerStore) ListAddrs(id p2p.PeerID) []string {
	m := map[string]struct{}{}
	for _, ps2 := range ps {
		for _, addr := range ps2.ListAddrs(id) {
			m[addr] = struct{}{}
		}
	}
	ret := make([]string, 0, len(m))
	for addr := range m {
		ret = append(ret, addr)
	}
	return ret
}

func (ps ChainPeerStore) Contains(id p2p.PeerID) bool {
	for _, ps2 := range ps {
		if ps2.Contains(id) {
			return true
		}
	}
	return false
}
