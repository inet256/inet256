package kadsrnet

import (
	"bytes"
	"context"
	sync "sync"

	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/pkg/errors"
)

type routeService struct {
	routeTable RouteTable
	sf         sendBodyAlongFunc

	mu      sync.Mutex
	queries map[Addr]*future
}

func newRouteService(routeTable RouteTable, sf sendBodyAlongFunc) *routeService {
	return &routeService{
		sf:         sf,
		routeTable: routeTable,
		queries:    make(map[Addr]*future),
	}
}

func (rs *routeService) queryRoutes(ctx context.Context, r *Route, q *QueryRoutes) (*RouteList, error) {
	dst := idFromBytes(r.Dst)
	rs.mu.Lock()
	fut, exists := rs.queries[dst]
	if !exists {
		fut = newFuture()
		rs.queries[dst] = fut
	}
	rs.mu.Unlock()
	defer rs.deleteQuery(dst)
	if !exists {
		body := &Body{
			Body: &Body_QueryRoutes{
				QueryRoutes: q,
			},
		}
		if err := rs.sf(ctx, dst, r, body); err != nil {
			return nil, err
		}
	}
	res, err := fut.get(ctx)
	if err != nil {
		return nil, err
	}
	return res.(*RouteList), nil
}

func (rs *routeService) deleteQuery(dst Addr) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.queries, dst)
}

func (rs *routeService) handleQueryRoutes(locus []byte, nbits int, limit int) (*RouteList, error) {
	routes, err := readRoutes(rs.routeTable, locus, nbits, limit)
	if err != nil {
		return nil, err
	}
	localAddr := rs.routeTable.LocalAddr()
	return &RouteList{
		Src:    localAddr[:],
		Routes: routes,
	}, nil
}

func (rs *routeService) handleRouteList(src Addr, rl *RouteList) error {
	if !bytes.Equal(src[:], rl.Src) {
		return errors.Errorf("route list src must match message src %x %x", src, rl.Src)
	}

	rs.mu.Lock()
	fut, exists := rs.queries[src]
	if !exists {
		rs.mu.Unlock()
		return errors.Errorf("got unsolicited routelist")
	}
	rs.mu.Unlock()

	fut.complete(rl)
	return nil
}

func readRoutes(rt RouteTable, locus []byte, nbits int, limit int) ([]*Route, error) {
	var stopIter = errors.New("stop iteration")
	var routes []*Route
	if err := rt.ForEach(func(r *Route) error {
		if kademlia.HasPrefix(r.Dst, locus, nbits) {
			routes = append(routes, r)
		}
		if len(routes) < limit {
			return nil
		}
		return stopIter
	}); err != nil && err != stopIter {
		return nil, err
	}
	return routes, nil
}

func containsRouteTo(routes []*Route, x Addr) bool {
	for _, r := range routes {
		if bytes.Equal(r.Dst, x[:]) {
			return true
		}
	}
	return false
}
