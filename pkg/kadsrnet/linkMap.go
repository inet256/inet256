package kadsrnet

import (
	"sync"

	"github.com/brendoncarroll/go-p2p"
)

type linkMap struct {
	mu     sync.RWMutex
	n      uint32
	id2Int map[p2p.PeerID]uint32
	int2ID map[uint32]p2p.PeerID
}

func newLinkMap() *linkMap {
	return &linkMap{
		id2Int: make(map[p2p.PeerID]uint32),
		int2ID: make(map[uint32]p2p.PeerID),
	}
}

func (lm *linkMap) ID2Int(id p2p.PeerID) uint32 {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	i, exists := lm.id2Int[id]
	if !exists {
		lm.n++
		i = lm.n
		lm.id2Int[id] = i
		lm.int2ID[i] = id
	}
	return i
}

func (lm *linkMap) Int2ID(i uint32) p2p.PeerID {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.int2ID[i]
}

func idFromBytes(x []byte) p2p.PeerID {
	id := p2p.PeerID{}
	copy(id[:], x)
	return id
}
