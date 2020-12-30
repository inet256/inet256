package kadsrnet

import (
	"bytes"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/inet256/inet256/pkg/inet256"
)

type PeerInfoStore struct {
	mu    sync.RWMutex
	cache *kademlia.Cache
}

func newPeerInfoStore(localID p2p.PeerID, size int) *PeerInfoStore {
	return &PeerInfoStore{
		cache: kademlia.NewCache(localID[:], size, 1),
	}
}

func (s *PeerInfoStore) Lookup(prefix []byte, nbits int) *PeerInfo {
	s.mu.RLock()
	e := s.cache.Closest(prefix)
	s.mu.RUnlock()
	if inet256.HasPrefix(e.Key, prefix, nbits) {
		return e.Value.(*PeerInfo)
	}
	return nil
}

func (s *PeerInfoStore) Get(target Addr) *PeerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.cache.Get(target[:])
	if v == nil {
		return nil
	}
	return v.(*PeerInfo)
}

func (s *PeerInfoStore) Add(pinfo *PeerInfo) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.cache.Put(pinfo.Id, pinfo)
	return !bytes.Equal(e.Key, pinfo.Id)
}
