package peers

import (
	"sync"

	"go.brendoncarroll.net/p2p"
	"go.inet256.org/inet256/pkg/inet256"
)

type peerStore[T p2p.Addr] struct {
	mu sync.RWMutex
	m  map[inet256.Addr]Info[T]
}

func NewStore[T p2p.Addr]() Store[T] {
	return &peerStore[T]{
		m: map[inet256.Addr]Info[T]{},
	}
}

func (s *peerStore[T]) Add(id inet256.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.m[id]; !exists {
		s.m[id] = Info[T]{}
	}
}

func (s *peerStore[T]) Remove(id inet256.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}

func (s *peerStore[T]) Get(id inet256.Addr) (Info[T], bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.m[id]
	return info, ok
}

func (s *peerStore[T]) Update(id inet256.Addr, fn func(Info[T]) Info[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.m[id]
	if ok {
		s.m[id] = fn(info)
	}
}

func (s *peerStore[T]) List() []inet256.Addr {
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

type ChainStore[T p2p.Addr] []Store[T]

func (ps ChainStore[T]) Add(inet256.Addr) {
	panic("cannot Add to ChainStore")
}

func (ps ChainStore[T]) Remove(inet256.Addr) {
	panic("cannot Remove from ChainStore")
}

func (ps ChainStore[T]) Get(id inet256.Addr) (Info[T], bool) {
	var ret Info[T]
	var exists bool
	for _, s := range ps {
		info, ok := s.Get(id)
		if ok {
			ret.Addrs = append(ret.Addrs, info.Addrs...)
			exists = true
		}
	}
	return ret, exists
}

func (ps ChainStore[T]) Update(x inet256.Addr, fn func(Info[T]) Info[T]) {
	panic("cannot Update on ChainStore")
}

func (ps ChainStore[T]) List() []inet256.Addr {
	m := map[inet256.Addr]struct{}{}
	for _, ps2 := range ps {
		for _, id := range ps2.List() {
			m[id] = struct{}{}
		}
	}
	ret := make([]inet256.Addr, 0, len(m))
	for id := range m {
		ret = append(ret, id)
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
