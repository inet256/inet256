package linkmap

import (
	"hash/fnv"
	"sync"

	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/inet256/inet256/pkg/inet256"
)

type LinkMap struct {
	locus  inet256.ID
	mu     sync.RWMutex
	id2Int map[inet256.ID]uint32
	int2ID map[uint32]inet256.ID
}

func New(locus inet256.ID) *LinkMap {
	return &LinkMap{
		locus:  locus,
		id2Int: make(map[inet256.ID]uint32),
		int2ID: make(map[uint32]inet256.ID),
	}
}

func (lm *LinkMap) ID2Int(id inet256.ID) uint32 {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if n, exists := lm.id2Int[id]; exists {
		return n
	}
	var xor [32]byte
	kademlia.XORBytes(xor[:], id[:], lm.locus[:])
	n := hash(xor[:])
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

func (lm *LinkMap) Int2ID(i uint32) inet256.ID {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.int2ID[i]
}

func hash(x []byte) uint32 {
	h := fnv.New32()
	h.Write(x[:])
	return h.Sum32()
}
