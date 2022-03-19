package peers

import (
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

type peerStore[T p2p.Addr] struct {
	mu sync.RWMutex
	m  map[inet256.Addr][]T
}

func NewStore[T p2p.Addr]() Store[T] {
	return &peerStore[T]{
		m: map[inet256.Addr][]T{},
	}
}

func (s *peerStore[T]) Add(id inet256.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.m[id]; !exists {
		s.m[id] = []T{}
	}
}

func (s *peerStore[T]) Remove(id inet256.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}

func (s *peerStore[T]) AddAddr(id inet256.Addr, addr T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	addrs := s.m[id]
	for i := range addrs {
		if addr.String() == addrs[i].String() {
			return
		}
	}
	s.m[id] = append(s.m[id], addr)
}

func (s *peerStore[T]) SetAddrs(id inet256.Addr, addrs []T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = addrs
}

func (s *peerStore[T]) ListPeers() []inet256.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := []inet256.Addr{}
	for id := range s.m {
		ids = append(ids, id)
	}
	return ids
}

func (s *peerStore[T]) Contains(id inet256.Addr) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.m[id]
	return exists
}

func (s *peerStore[T]) ListAddrs(id inet256.Addr) []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[id]
}

type ChainStore[T p2p.Addr] []Store[T]

func (ps ChainStore[T]) Add(inet256.Addr) {
	panic("cannot Add to ChainStore")
}

func (ps ChainStore[T]) Remove(inet256.Addr) {
	panic("cannot Remove from ChainStore")
}

func (ps ChainStore[T]) SetAddrs(x inet256.Addr, addrs []T) {
	panic("cannot SetAddrs on ChainStore")
}

func (ps ChainStore[T]) ListPeers() []inet256.Addr {
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

func (ps ChainStore[T]) ListAddrs(id inet256.Addr) []T {
	m := map[string]struct{}{}
	ret := make([]T, 0, len(m))
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

func (ps ChainStore[T]) Contains(id inet256.Addr) bool {
	for _, ps2 := range ps {
		if ps2.Contains(id) {
			return true
		}
	}
	return false
}
