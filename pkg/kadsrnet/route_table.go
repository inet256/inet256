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
	localID Addr
	mu      sync.RWMutex
	cache   *kademlia.Cache
}

func NewRouteTable(localID p2p.PeerID, size int) *RouteTable {
	return &RouteTable{
		localID: localID,
		cache:   kademlia.NewCache(localID[:], size, 1),
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
	p := ent.Value.(Path)
	id := p2p.PeerID{}
	copy(id[:], prefix)
	return &Route{
		Dst:  id[:],
		Path: p,
	}
}

func (rt *RouteTable) IsFull() bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cache.IsFull()
}

func (rt *RouteTable) WouldAdd(r *Route) bool {
	d := [32]byte{}
	kademlia.XORBytes(d[:], rt.localID[:], r.Dst)
	return kademlia.Leading0s(d[:]) >= rt.MustMatchBits()
}

func (rt *RouteTable) MustMatchBits() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cache.WouldAccept()
}

func (rt *RouteTable) ListPeers() (ret []p2p.PeerID) {
	rt.ForEach(func(r *Route) {
		id := p2p.PeerID{}
		copy(id[:], r.GetDst())
		ret = append(ret, id)
	})
	return ret
}

func (rt *RouteTable) Contains(addr Addr) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	v := rt.cache.Get(addr[:])
	return v != nil
}

func (rt *RouteTable) ForEach(fn func(*Route)) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	rt.cache.ForEach(func(e kademlia.Entry) bool {
		r := e.Value.(*Route)
		fn(r)
		return true
	})
}

func (rt *RouteTable) LocalID() Addr {
	return rt.localID
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
