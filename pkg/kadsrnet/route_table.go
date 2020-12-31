package kadsrnet

import (
	"bytes"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/inet256/inet256/pkg/inet256"
)

type Path = []uint32

var _ inet256.PeerSet = &RouteTable{}

type RouteTable struct {
	mu    sync.RWMutex
	cache *kademlia.Cache
}

func NewRouteTable(localID p2p.PeerID, size int) *RouteTable {
	return &RouteTable{
		cache: kademlia.NewCache(localID[:], size, 1),
	}
}

// Add returns true if the route was accepted
func (rt *RouteTable) Add(r *Route) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.add(r)
}

func (rt *RouteTable) Get(dst Addr) *Route {
	return rt.get(dst[:])
}

func (rt *RouteTable) Delete(dst []byte) bool {
	e := rt.cache.Delete(dst)
	return e != nil
}

func (rt *RouteTable) add(r *Route) bool {
	e := rt.cache.Put(r.Dst, r)
	if e == nil {
		return true
	}
	return !bytes.Equal(e.Key, r.Dst)
}

func (rt *RouteTable) get(dst []byte) *Route {
	v := rt.cache.Get(dst)
	if v == nil {
		return nil
	}
	return v.(*Route)
}

func (rt *RouteTable) Closest(prefix []byte) *Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	ent := rt.cache.Closest(prefix)
	if ent == nil {
		return nil
	}
	return ent.Value.(*Route)
}

func (rt *RouteTable) IsFull() bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cache.IsFull()
}

func (rt *RouteTable) WouldAdd(r *Route) bool {
	rt.mu.RLock()
	rt.mu.RUnlock()
	return rt.cache.WouldAdd(r.Dst)
}

func (rt *RouteTable) MustMatchBits() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cache.AcceptingPrefixLen()
}

func (rt *RouteTable) ListPeers() (ret []p2p.PeerID) {
	rt.ForEach(func(r *Route) bool {
		id := p2p.PeerID{}
		copy(id[:], r.GetDst())
		ret = append(ret, id)
		return true
	})
	return ret
}

func (rt *RouteTable) Contains(addr Addr) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	v := rt.cache.Get(addr[:])
	return v != nil
}

func (rt *RouteTable) ForEach(fn func(*Route) bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	rt.cache.ForEach(func(e kademlia.Entry) bool {
		r := e.Value.(*Route)
		return fn(r)
	})
}

func (rt *RouteTable) LocalID() Addr {
	return idFromBytes(rt.cache.Locus())
}

func ConcatRoutes(left, right *Route) *Route {
	if right == nil {
		return left
	}
	if left == nil {
		return right
	}
	p := Path{}
	p = append(p, left.Path...)
	p = append(p, right.Path...)
	timestamp := left.Timestamp
	if right.Timestamp < timestamp {
		timestamp = right.Timestamp
	}
	return &Route{
		Dst:       right.Dst,
		Path:      p,
		Timestamp: timestamp,
	}
}

func compareDistance(target []byte, a, b []byte) int {
	aDist := [32]byte{}
	kademlia.XORBytes(aDist[:], target, a)
	bDist := [32]byte{}
	kademlia.XORBytes(bDist[:], target, b)
	return bytes.Compare(aDist[:], bDist[:])
}
