package kadsrnet

import (
	"context"
	"log"
	sync "sync"
	"time"

	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const crawlerPingTimeout = 10 * time.Second

type sendQueryFunc = func(ctx context.Context, dst Addr, q *QueryRoutes) error

// crawler is responsible for populating the route table
type crawler struct {
	routeTable RouteTable
	pinger     *pinger
	infoSrv    *infoService
	sqf        sendQueryFunc

	mu           sync.Mutex
	peerMonitors map[Addr]struct{}
}

func newCrawler(rt RouteTable, pinger *pinger, infoSrv *infoService, sqf sendQueryFunc) *crawler {
	return &crawler{
		routeTable:   rt,
		pinger:       pinger,
		infoSrv:      infoSrv,
		sqf:          sqf,
		peerMonitors: make(map[Addr]struct{}),
	}
}

func (c *crawler) onRoutes(ctx context.Context, from Addr, routes []*Route) error {
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
		if !c.routeTable.WouldAdd(r2) {
			continue
		}
		if len(r2.Path) > MaxPathLen {
			continue
		}
		eg.Go(func() error {
			// ensure we have their info
			_, err := c.infoSrv.get(ctx, idFromBytes(r2.Dst), r2.Path)
			if err != nil {
				return err
			}
			// ensure the route works
			_, err = c.pinger.ping(ctx, idFromBytes(r2.Dst), r2.Path)
			if err != nil {
				log.Println("ping error", err)
				return err
			}
			log.Println("confirmed ping")
			mu.Lock()
			confirmedRoutes = append(confirmedRoutes, r2)
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		if err != context.Canceled {
			return err
		}
	}
	log.Printf("found %d routes from %v", len(confirmedRoutes), from)
	for _, r := range confirmedRoutes {
		if c.routeTable.Add(r) {
			dst := idFromBytes(r.Dst)
			if err := c.crawlSingle(ctx, dst); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *crawler) onQueryRoutes(locus []byte, nbits int, limit int) (*RouteList, error) {
	var stopIter = errors.New("stop iteration")
	var routes []*Route
	if err := c.routeTable.ForEach(func(r *Route) error {
		if kademlia.HasPrefix(r.Dst, locus[:], nbits) {
			routes = append(routes, r)
		} else {
			log.Println("no match", r.Dst, locus, nbits)
		}
		if len(routes) < limit {
			return nil
		}
		return stopIter
	}); err != nil && err != stopIter {
		return nil, err
	}
	localAddr := c.routeTable.LocalAddr()
	return &RouteList{
		Src:    localAddr[:],
		Routes: routes,
	}, nil
}

func (c *crawler) run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	lastRun := time.Now().Unix()
	for {
		thisRun := time.Now().Unix()
		if err := c.crawlAll(ctx, lastRun); err != nil {
			logrus.Error("error crawling: ", err)
		}
		lastRun = thisRun
		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
}

func (c *crawler) crawlAll(ctx context.Context, lastRun int64) (retErr error) {
	c.routeTable.ForEach(func(r *Route) error {
		dst := idFromBytes(r.Dst)
		err := c.crawlSingle(ctx, dst)
		if retErr == nil {
			retErr = err
		}
		return nil
	})
	return retErr
}

func (c *crawler) crawlSingle(ctx context.Context, dst Addr) error {
	// TODO: this is a leaky abstraction
	nbits := c.routeTable.(*routeTable).cache.(*kadRouteTable).MustMatchBits()
	localAddr := c.routeTable.LocalAddr()
	return c.sqf(ctx, dst, &QueryRoutes{
		Locus: localAddr[:],
		Nbits: uint32(nbits),
		Limit: 10,
	})
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

// monitorRoute pings dst, using path in a loop, and returns when it stops
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
