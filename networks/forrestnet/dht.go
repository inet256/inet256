package forrestnet

import (
	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/inet256/inet256/pkg/inet256"
)

type DHT[V any] struct {
	cache *kademlia.Cache[V]
}

func NewDHT[V any](locus inet256.ID, size int) *DHT[V] {
	return &DHT[V]{
		cache: kademlia.NewCache[V](locus[:], size, 1),
	}
}
