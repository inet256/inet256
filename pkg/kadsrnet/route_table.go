package kadsrnet

import (
	"bytes"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/inet256/inet256/pkg/inet256"
)

type Path = []uint32

type RouteTable interface {
	Add(r *Route) bool
	// Get returns a route to dst or nil
	Get(dst Addr) *Route
	// Delete deletes the route to dst and returns true if it existed and was deleted
	Delete(dst Addr) bool
	Update(dst Addr, fn func(r *Route) *Route) bool
	// RouteTo differs from Get in that it does not have to return a route to dst.
	RouteTo(dst Addr) *Route

	ForEach(func(*Route) error) error

	LocalAddr() Addr
	WouldAdd(r *Route) bool
}

// this is the top routeTable which knows how to hand the special case one hop peers and
// also the local addr.
type routeTable struct {
	localAddr   Addr
	oneHopPeers inet256.PeerSet
	linkMap     *linkMap
	cache       RouteTable

	mu               sync.RWMutex
	oneHopTimestamps map[Addr]int64
}

func newRouteTable(localAddr Addr, oneHopPeers inet256.PeerSet, lm *linkMap, cache RouteTable) *routeTable {
	return &routeTable{
		localAddr:   localAddr,
		linkMap:     lm,
		oneHopPeers: oneHopPeers,
		cache:       cache,
	}
}

func (rt *routeTable) Add(r *Route) bool {
	dst := idFromBytes(r.Dst)
	switch {
	case dst == rt.localAddr:
		return false
	case rt.oneHopPeers.Contains(dst):
		return false
	default:
		return rt.cache.Add(r)
	}
}

func (rt *routeTable) Get(dst Addr) *Route {
	switch {
	case dst == rt.localAddr:
		return nil
	case rt.oneHopPeers.Contains(dst):
		linkIndex := rt.linkMap.ID2Int(dst)
		if linkIndex < 1 {
			panic(linkIndex)
		}
		rt.mu.RLock()
		timestamp := rt.oneHopTimestamps[dst]
		rt.mu.RUnlock()
		return &Route{
			Dst:       dst[:],
			Path:      Path{linkIndex},
			Timestamp: timestamp,
		}
	default:
		return rt.cache.Get(dst)
	}
}

func (rt *routeTable) Update(dst Addr, fn func(*Route) *Route) bool {
	switch {
	case dst == rt.localAddr:
		return false
	case rt.oneHopPeers.Contains(dst):
		r := fn(rt.Get(dst))
		if r == nil {
			return false
		}
		rt.mu.Lock()
		rt.oneHopTimestamps[dst] = r.Timestamp
		rt.mu.Unlock()
		return true
	default:
		return rt.cache.Update(dst, fn)
	}
}

func (rt *routeTable) LocalAddr() Addr {
	return rt.localAddr
}

func (rt *routeTable) Delete(dst Addr) bool {
	switch {
	case dst == rt.localAddr:
		return false
	case rt.oneHopPeers.Contains(dst):
		return false
	default:
		return rt.cache.Delete(dst)
	}
}

func (rt *routeTable) RouteTo(dst Addr) *Route {
	switch {
	case dst == rt.localAddr:
		return rt.Get(dst)
	case rt.oneHopPeers.Contains(dst):
		return rt.Get(dst)
	default:
		return rt.cache.RouteTo(dst)
	}
}

func (rt *routeTable) WouldAdd(r *Route) bool {
	dst := idFromBytes(r.Dst)
	switch {
	case dst == rt.localAddr:
		return false
	case rt.oneHopPeers.Contains(dst):
		return false
	default:
		return rt.cache.WouldAdd(r)
	}
}

func (rt *routeTable) ForEach(fn func(*Route) error) error {
	// one hope peers
	for _, peerID := range rt.oneHopPeers.ListPeers() {
		if err := fn(rt.Get(peerID)); err != nil {
			return err
		}
	}
	return rt.cache.ForEach(fn)
}

type kadRouteTable struct {
	mu    sync.RWMutex
	cache *kademlia.Cache
}

func newKadRouteTable(localID p2p.PeerID, size int) *kadRouteTable {
	return &kadRouteTable{
		cache: kademlia.NewCache(localID[:], size, 1),
	}
}

// Add returns true if the route was accepted
func (rt *kadRouteTable) Add(r *Route) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	current := rt.get(r.Dst)
	return rt.put(BestRoute(current, r))
}

func (rt *kadRouteTable) Get(dst Addr) *Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.get(dst[:])
}

func (rt *kadRouteTable) Delete(dst Addr) bool {
	e := rt.cache.Delete(dst[:])
	return e != nil
}

func (rt *kadRouteTable) put(r *Route) bool {
	e := rt.cache.Put(r.Dst, r)
	if e == nil {
		return true
	}
	return !bytes.Equal(e.Key, r.Dst)
}

func (rt *kadRouteTable) get(dst []byte) *Route {
	v := rt.cache.Get(dst)
	if v == nil {
		return nil
	}
	return v.(*Route)
}

func (rt *kadRouteTable) RouteTo(dst Addr) *Route {
	return rt.Closest(dst[:])
}

func (rt *kadRouteTable) Closest(prefix []byte) *Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	ent := rt.cache.Closest(prefix)
	if ent == nil {
		return nil
	}
	return ent.Value.(*Route)
}

func (rt *kadRouteTable) Update(dst Addr, fn func(*Route) *Route) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	x := rt.get(dst[:])
	if x == nil {
		return false
	}
	y := fn(x)
	if !bytes.Equal(x.Dst, y.Dst) {
		panic("update cannot change dst")
	}
	rt.put(y)
	return true
}

func (rt *kadRouteTable) IsFull() bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cache.IsFull()
}

func (rt *kadRouteTable) WouldAdd(r *Route) bool {
	rt.mu.RLock()
	rt.mu.RUnlock()
	if current := rt.get(r.Dst); current != nil {
		return BestRoute(current, r) == r
	}
	return rt.cache.WouldAdd(r.Dst)
}

func (rt *kadRouteTable) MustMatchBits() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.cache.AcceptingPrefixLen()
}

func (rt *kadRouteTable) Contains(addr Addr) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	v := rt.cache.Get(addr[:])
	return v != nil
}

func (rt *kadRouteTable) ForEach(fn func(*Route) error) (retErr error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	rt.cache.ForEach(func(e kademlia.Entry) bool {
		r := e.Value.(*Route)
		if retErr = fn(r); retErr != nil {
			return false
		}
		return true
	})
	return retErr
}

func (rt *kadRouteTable) LocalAddr() Addr {
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

func BestRoute(a, b *Route) *Route {
	switch {
	case a != nil && b != nil && !bytes.Equal(a.Dst, b.Dst):
		panic("incomparable routes")
	case a == nil:
		return b
	case b == nil:
		return a
	case a == nil && b == nil:
		return nil
	case len(b.Path) < len(a.Path):
		return b
	default:
		return a
	}
}

func compareDistance(target []byte, a, b []byte) int {
	aDist := dist32(target, a)
	bDist := dist32(target, b)
	return bytes.Compare(aDist[:], bDist[:])
}

func dist32(a, b []byte) [32]byte {
	if len(a) > 32 {
		panic(a)
	}
	if len(b) > 32 {
		panic(b)
	}
	d := [32]byte{}
	kademlia.XORBytes(d[:], a, b)
	return d
}
