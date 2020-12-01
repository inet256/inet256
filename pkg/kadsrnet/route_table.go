package kadsrnet

import (
	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
)

type Path []uint32

type Route struct {
	Dst  Addr
	Path Path
}

var _ inet256.PeerSet = &RouteTable{}

type RouteTable struct {
	mu sync.Mutex
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
	evicted := rt.cache.Put(r.Dst[:], r.Path)
	return !bytes.Equal(evicted.Key, r.Dst[:])
}

func (rt *RouteTable) AddRelativeTo(r Route, from p2p.PeerID) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	baseRoute := rt.cache.Get(r)
	if baseRoute == nil{
		return false
	}
	r.Path = append(baseRoute.Path, r.Path...)
	return rt.add(r)
}

func (rt *RouteTable) get(src p2p.PeerID) *Route {
	e := rt.cache.Lookup(src[:])
	if e == nil {
		return nil
	}
	r := e.(Route)
	return &r
}

func (rt *RouteTable) Closest(prefix []byte) Route {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	ent := rt.cache.Closest(prefix)
	p := ent.Value.(Path)
	id := p2p.PeerID{}
	copy(id[:], prefix)
	return Route{
		Dst: id,
		Path: p
	}
}

func (rt *RouteTable) IsFull() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.cache.IsFull()
}

func (rt *RouteTable) WouldAccept() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.cache.WouldAccept()
}

func (rt *RouteTable) ListPeers() (ret []p2p.PeerID) {	
	rt.ForEach(func(r Route) {
		ret = append(ret, r)
	})
}

func (rt *RouteTable) ForEach(fn func(Route)) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.cache.ForEach(func(e *kademlia.Entry) {
		dst := p2p.PeerID{}
		copy(dst[:], e.Key)
		p := e.Value.(Path)
		r := Route{
			Dst: dst,
			Path: p,
		}
		fn(r)
	})
}
