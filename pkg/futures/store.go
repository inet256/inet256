package futures

import (
	"errors"
	"sync"
)

var errFutureCancelled = errors.New("future cancelled")

type Store[K comparable, V any] struct {
	mu sync.Mutex
	m  map[K]*Future[V]
}

func NewStore[K comparable, V any]() *Store[K, V] {
	return &Store[K, V]{
		m: make(map[K]*Future[V]),
	}
}

func (s *Store[K, V]) Get(k K) *Future[V] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[k]
}

func (s *Store[K, V]) GetOrCreate(k K) (*Future[V], bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fut, exists := s.m[k]
	if !exists {
		fut = New[V]()
		s.m[k] = fut
	}
	return fut, !exists
}

func (s *Store[K, V]) Delete(k K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fut, exists := s.m[k]; exists {
		fut.Fail(errFutureCancelled)
	}
	delete(s.m, k)
}

func (s *Store[K, V]) ForEach(fn func(k K, v *Future[V]) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.m {
		if !fn(k, v) {
			break
		}
	}
}
