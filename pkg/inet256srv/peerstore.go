package inet256srv

import (
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

type peerStore struct {
	mu sync.RWMutex
	m  map[inet256.Addr][]p2p.Addr
}

func NewPeerStore() inet256.PeerStore {
	return &peerStore{
		m: map[inet256.Addr][]p2p.Addr{},
	}
}

func (s *peerStore) Add(id inet256.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.m[id]; !exists {
		s.m[id] = []p2p.Addr{}
	}
}

func (s *peerStore) Remove(id inet256.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}

func (s *peerStore) AddAddr(id inet256.Addr, addr p2p.Addr) {
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

func (s *peerStore) SetAddrs(id inet256.Addr, addrs []p2p.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = addrs
}

func (s *peerStore) ListPeers() []inet256.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := []inet256.Addr{}
	for id := range s.m {
		ids = append(ids, id)
	}
	return ids
}

func (s *peerStore) Contains(id inet256.Addr) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.m[id]
	return exists
}

func (s *peerStore) ListAddrs(id inet256.Addr) []p2p.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[id]
}

var _ PeerStore = ChainPeerStore{}

type ChainPeerStore []PeerStore

func (ps ChainPeerStore) Add(inet256.Addr) {
	panic("cannot Add to ChainPeerStore")
}

func (ps ChainPeerStore) Remove(inet256.Addr) {
	panic("cannot Remove from ChainPeerStore")
}

func (ps ChainPeerStore) SetAddrs(x inet256.Addr, addrs []p2p.Addr) {
	panic("cannot SetAddrs on ChainPeerStore")
}

func (ps ChainPeerStore) ListPeers() []inet256.Addr {
	m := map[inet256.Addr]struct{}{}
	for _, ps2 := range ps {
		for _, id := range ps2.ListPeers() {
			m[id] = struct{}{}
		}
	}
	ret := make([]inet256.Addr, 0, len(m))
	for id := range m {
		ret = append(ret, id)
	}
	return ret
}

func (ps ChainPeerStore) ListAddrs(id inet256.Addr) []p2p.Addr {
	m := map[string]struct{}{}
	ret := make([]p2p.Addr, 0, len(m))
	for _, ps2 := range ps {
		for _, addr := range ps2.ListAddrs(id) {
			key := addr.String()
			if _, exists := m[key]; exists {
				continue
			}
			m[key] = struct{}{}
			ret = append(ret, addr)
		}
	}
	return ret
}

func (ps ChainPeerStore) Contains(id inet256.Addr) bool {
	for _, ps2 := range ps {
		if ps2.Contains(id) {
			return true
		}
	}
	return false
}
