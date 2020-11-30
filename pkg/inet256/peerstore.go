package inet256

import (
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/sirupsen/logrus"
)

type peerStore struct {
	mu sync.RWMutex
	m  map[p2p.PeerID][]string
}

func NewPeerStore() *peerStore {
	return &peerStore{
		m: map[p2p.PeerID][]string{},
	}
}

func (s *peerStore) AddPeer(id p2p.PeerID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.m[id]; !exists {
		s.m[id] = []string{}
	}
}

func (s *peerStore) AddAddr(id p2p.PeerID, addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = append(s.m[id], addr)
}

func (s *peerStore) PutAddrs(id p2p.PeerID, addrs []string) {
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

func (s *peerStore) ListAddrs(id p2p.PeerID) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[id]
}

type addrSource struct {
	swarm p2p.Swarm
	store PeerStore
}

func newAddrSource(swarm p2p.Swarm, store PeerStore) peerswarm.AddrSource {
	return func(id p2p.PeerID) []p2p.Addr {
		xs := store.ListAddrs(id)
		var ys []p2p.Addr
		for i := range xs {
			y, err := swarm.ParseAddr([]byte(xs[i]))
			if err != nil {
				logrus.Error("error parsing addr:", err)
				continue
			}
			ys = append(ys, y)
		}
		return ys
	}
}
