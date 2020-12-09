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
func (rt *RouteTable) Add(r Route) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.add(r)
}

func (rt *RouteTable) add(r Route) bool {
	e := rt.cache.Put(r.Dst, r)
	return !bytes.Equal(e.Key, r.Dst)
}

func (rt *RouteTable) get(dst []byte) *Route {
	v := rt.cache.Lookup(dst)
	if v == nil {
		return nil
	}
	return v.(*Route)
}

func (rt *RouteTable) AddRelativeTo(r Route, from p2p.PeerID) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	baseRoute := rt.get(r.GetDst())
	if baseRoute == nil {
		return false
	}
	r.Path = append(baseRoute.Path, r.Path...)
	return rt.add(r)
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

func (rt *RouteTable) WouldAccept() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cache.WouldAccept()
}

func (rt *RouteTable) ListPeers() (ret []p2p.PeerID) {
	rt.ForEach(func(r Route) {
		id := p2p.PeerID{}
		copy(id[:], r.GetDst())
		ret = append(ret, id)
	})
	return ret
}

func (rt *RouteTable) Contains(addr Addr) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	v := rt.cache.Lookup(addr[:])
	return v != nil
}

func (rt *RouteTable) ForEach(fn func(Route)) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	rt.cache.ForEach(func(e kademlia.Entry) bool {
		p := e.Value.(Path)
		r := Route{
			Dst:  e.Key,
			Path: p,
		}
		fn(r)
		return true
	})
}
