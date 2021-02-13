package kadsrnet

import (
	"bytes"
	"context"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
)

type PeerInfoStore struct {
	localAddr   Addr
	publicKey   p2p.PublicKey
	swarm       peerswarm.Swarm
	oneHopPeers inet256.PeerSet

	mu    sync.RWMutex
	cache *kademlia.Cache
}

func newPeerInfoStore(publicKey p2p.PublicKey, swarm peerswarm.Swarm, oneHopPeers inet256.PeerSet, size int) *PeerInfoStore {
	localAddr := p2p.NewPeerID(publicKey)
	return &PeerInfoStore{
		localAddr:   localAddr,
		publicKey:   publicKey,
		swarm:       swarm,
		oneHopPeers: oneHopPeers,
		cache:       kademlia.NewCache(localAddr[:], size, 1),
	}
}

func (s *PeerInfoStore) Lookup(prefix []byte, nbits int) *PeerInfo {
	if kademlia.HasPrefix(s.localAddr[:], prefix, nbits) {
		return s.Get(s.localAddr)
	}
	for _, peerID := range s.oneHopPeers.ListPeers() {
		if kademlia.HasPrefix(peerID[:], prefix, nbits) {
			return s.Get(peerID)
		}
	}
	s.mu.RLock()
	e := s.cache.Closest(prefix)
	s.mu.RUnlock()
	if e != nil && inet256.HasPrefix(e.Key, prefix, nbits) {
		return e.Value.(*PeerInfo)
	}
	return nil
}

func (s *PeerInfoStore) Get(target Addr) *PeerInfo {
	if s.localAddr == target {
		return &PeerInfo{
			PublicKey: p2p.MarshalPublicKey(s.publicKey),
		}
	}
	if s.oneHopPeers.Contains(target) {
		publicKey, err := s.swarm.LookupPublicKey(context.TODO(), target)
		if err != nil {
			return nil
		}
		return &PeerInfo{
			PublicKey: p2p.MarshalPublicKey(publicKey),
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.cache.Get(target[:])
	if v == nil {
		return nil
	}
	return v.(*PeerInfo)
}

func (s *PeerInfoStore) Put(x Addr, pinfo *PeerInfo) bool {
	if s.localAddr == x {
		return false
	}
	publicKey, err := p2p.ParsePublicKey(pinfo.PublicKey)
	if err != nil {
		return false
	}
	peerID := p2p.NewPeerID(publicKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.cache.Put(peerID[:], pinfo)
	return e == nil || !bytes.Equal(e.Key, peerID[:])
}
