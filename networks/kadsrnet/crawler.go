package kadsrnet

import (
	"context"
	sync "sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const crawlerPingTimeout = 10 * time.Second

type sendBodyAlongFunc = func(ctx context.Context, dst Addr, r *Route, b *Body) error

// crawler is responsible for populating the route table
type crawler struct {
	routeTable RouteTable
	pinger     *pinger
	infoSrv    *infoService
	routeSrv   *routeService
	log        *logrus.Logger

	mu           sync.Mutex
	peerMonitors map[Addr]struct{}
}

func newCrawler(rt RouteTable, routeSrv *routeService, pinger *pinger, infoSrv *infoService, log *logrus.Logger) *crawler {
	return &crawler{
		routeTable:   rt,
		pinger:       pinger,
		infoSrv:      infoSrv,
		routeSrv:     routeSrv,
		log:          log,
		peerMonitors: make(map[Addr]struct{}),
	}
}

// confirmRoutes first ensures we have the peer info for the peer, and then sends
// a ping along the route to confirm that is working
func (c *crawler) confirmRoutes(ctx context.Context, from Addr, routes []*Route) ([]*Route, error) {
	ctx, cf := context.WithTimeout(ctx, crawlerPingTimeout)
	defer cf()
	mu := sync.Mutex{}
	var confirmedRoutes []*Route
	eg := errgroup.Group{}
	for _, r := range routes {
		baseRoute := c.routeTable.Get(from)
		// if we don't have a route to the peer that sent it, ignore
		if baseRoute == nil {
			continue
		}
		r2 := ConcatRoutes(baseRoute, r)
		if len(r2.Path) > MaxPathLen {
			continue
		}
		if !c.routeTable.WouldAdd(r2) {
			continue
		}
		eg.Go(func() error {
			if err := c.confirmRoute(ctx, r2); err != nil {
				return nil
			}
			mu.Lock()
			confirmedRoutes = append(confirmedRoutes, r2)
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		if err != context.Canceled {
			return nil, err
		}
	}
	return confirmedRoutes, nil
}

func (c *crawler) confirmRoute(ctx context.Context, r *Route) error {
	// send our info.
	// ensure we have their info
	_, err := c.infoSrv.get(ctx, idFromBytes(r.Dst), r)
	if err != nil {
		return err
	}
	// ensure the route works
	_, err = c.pinger.ping(ctx, idFromBytes(r.Dst), r.Path)
	if err != nil {
		return err
	}
	return nil
}

func (c *crawler) run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		if err := c.crawlAll(ctx); err != nil {
			logrus.Error("error crawling: ", err)
		}
	}
}

func (c *crawler) crawlAll(ctx context.Context) (retErr error) {
	var routes []*Route
	c.routeTable.ForEach(func(r *Route) error {
		routes = append(routes, r)
		return nil
	})
	for _, r := range routes {
		err := c.crawlSingle(ctx, r)
		if retErr == nil {
			retErr = err
		}
	}
	return retErr
}

// crawlSingle starts crawling from a peer that has already been confirmed.
func (c *crawler) crawlSingle(ctx context.Context, r *Route) error {
	dst := idFromBytes(r.Dst)
	if currentRoute := c.routeTable.Get(dst); BestRoute(r, currentRoute) == currentRoute {
		return nil
	}
	// TODO: this is a leaky abstraction
	nbits := c.routeTable.(*routeTable).cache.(*kadRouteTable).MustMatchBits()
	localAddr := c.routeTable.LocalAddr()
	routeList, err := c.routeSrv.queryRoutes(ctx, r, &QueryRoutes{
		Locus: localAddr[:],
		Nbits: uint32(nbits),
		Limit: 10,
	})
	if err != nil {
		return err
	}
	confirmedRoutes, err := c.confirmRoutes(ctx, idFromBytes(routeList.Src), routeList.Routes)
	if err != nil {
		return err
	}
	for _, r := range confirmedRoutes {
		c.addRoute(r)
	}
	eg := errgroup.Group{}
	for _, r := range confirmedRoutes {
		r := r
		eg.Go(func() error {
			return c.crawlSingle(ctx, r)
		})
	}
	return eg.Wait()
}

// investigate starts crawling a peer, but confirms their route first.
// it should be called after getting a query from a peer.
func (c *crawler) investigate(ctx context.Context, r *Route) error {
	if !c.routeTable.WouldAdd(r) {
		return nil
	}
	if err := c.confirmRoute(ctx, r); err != nil {
		return err
	}
	c.addRoute(r)
	return c.crawlSingle(ctx, r)
}

func (c *crawler) addRoute(r *Route) {
	if c.routeTable.Add(r) {
		dst := idFromBytes(r.Dst)
		c.spawnMonitor(dst)
	}
}

func (c *crawler) spawnMonitor(dst Addr) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.peerMonitors[dst]
	if exists {
		return
	}
	c.peerMonitors[dst] = struct{}{}
	go func() {
		defer c.deleteMonitor(dst)
		ctx := context.TODO()
		c.monitorPeer(ctx, dst)
	}()
}

func (c *crawler) deleteMonitor(x Addr) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.peerMonitors, x)
}

// monitorPeer pings dst, using path in a loop, and returns when it stops
// responding.
// when route monitor returns the entry is cleared from the cache
func (c *crawler) monitorPeer(ctx context.Context, dst Addr) {
	defer c.routeTable.Delete(dst)
	ticker := time.NewTicker(crawlerPingTimeout)
	defer ticker.Stop()
	for {
		r := c.routeTable.Get(dst)
		if r == nil {
			// if they were evicted, then shutdown
			return
		}
		err := func() error {
			ctx, cf := context.WithTimeout(ctx, crawlerPingTimeout)
			defer cf()
			_, err := c.pinger.ping(ctx, dst, r.Path)
			return err
		}()
		if err != nil {
			// if they are unreachable, shutdown
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
