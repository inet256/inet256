package forrestnet

import (
	"sync"

	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/inet256/inet256/pkg/inet256"
)

type NodeInfo struct {
	PublicKey inet256.PublicKey
	Locators  map[inet256.Addr]Locator
}

type Locator struct {
	Timestamp Timestamp
	Path      []uint32
}

// type InfoClient struct {
// }

// func (c *InfoClient) Find(ctx context.Context) (*NodeInfo, error) {
// 	return nil, nil
// }

// type InfoServer struct {
// 	oneHop  peers.Set
// 	localID inet256.Addr
// 	store   *InfoStore
// }

// func NewInfoServer(oneHop peers.Set, localID inet256.Addr) *InfoServer {
// 	return &InfoServer{
// 		oneHop:  oneHop,
// 		localID: localID,
// 		store:   NewInfoStore(localID, 128),
// 	}
// }

type InfoStore struct {
	mu    sync.RWMutex
	cache *kademlia.Cache[*NodeInfo]
}

func NewInfoStore(localID inet256.Addr, size int) *InfoStore {
	return &InfoStore{
		cache: kademlia.NewCache[*NodeInfo](localID[:], size, 1),
	}
}

func (s *InfoStore) Get(x inet256.Addr) *NodeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ni, _ := s.cache.Get(x[:])
	return ni
}

func (s *InfoStore) Put(treeRoot inet256.Addr, pubKey inet256.PublicKey, loc Locator) {
	addr := inet256.NewAddr(pubKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.update(addr, func(x *NodeInfo) *NodeInfo {
		if x == nil {
			x = &NodeInfo{
				PublicKey: pubKey,
				Locators:  make(map[inet256.Addr]Locator),
			}
		}
		x.Locators[treeRoot] = loc
		return x
	})
}

func (s *InfoStore) update(addr inet256.Addr, fn func(x *NodeInfo) *NodeInfo) {
	x, _ := s.cache.Get(addr[:])
	y := fn(x)
	s.cache.Put(addr[:], y)
}

// Find returns the PeerInfo closest to x.
func (s *InfoStore) Find(x inet256.Addr) *NodeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ent := s.cache.Closest(x[:])
	if ent == nil {
		return nil
	}
	return ent.Value
}
