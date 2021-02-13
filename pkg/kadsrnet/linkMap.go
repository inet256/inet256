package kadsrnet

import (
	"hash/fnv"
	"sync"

	"github.com/brendoncarroll/go-p2p"
)

type linkMap struct {
	mu     sync.RWMutex
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
	if n, exists := lm.id2Int[id]; exists {
		return n
	}
	h := fnv.New32()
	h.Write(id[:])
	n := h.Sum32()
	for {
		if n == 0 {
			n++
			continue
		}
		_, exists := lm.int2ID[n]
		if !exists {
			lm.int2ID[n] = id
			lm.id2Int[id] = n
			return n
		}
		n++
	}
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
