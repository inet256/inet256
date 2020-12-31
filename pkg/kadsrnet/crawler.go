package kadsrnet

import (
	"bytes"
	"context"
	sync "sync"
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const crawlerPingTimeout = 10 * time.Second

type sendQueryFunc = func(ctx context.Context, dst Addr, q *QueryRoutes) error

// crawler is responsible for populating the route table
type crawler struct {
	oneHopPeers inet256.PeerSet
	routeTable  *RouteTable
	linkMap     *linkMap
	pinger      *pinger
	sqf         sendQueryFunc

	mu           sync.Mutex
	peerMonitors map[Addr]struct{}
}

func newCrawler(rt *RouteTable, oneHopPeers inet256.PeerSet, lm *linkMap, pinger *pinger, sqf sendQueryFunc) *crawler {
	return &crawler{
		routeTable:   rt,
		oneHopPeers:  oneHopPeers,
		linkMap:      lm,
		pinger:       pinger,
		sqf:          sqf,
		peerMonitors: make(map[Addr]struct{}),
	}
}

func (c *crawler) handleRoutes(ctx context.Context, from Addr, routes []*Route) error {
	ctx, cf := context.WithTimeout(ctx, crawlerPingTimeout)
	defer cf()
	mu := sync.Mutex{}
	var confirmedRoutes []*Route
	eg := errgroup.Group{}
	for _, r := range routes {
		dst := idFromBytes(r.Dst)
		// if it is a route to us, ignore
		if localID := c.routeTable.LocalID(); bytes.Equal(r.Dst, localID[:]) {
			continue
		}
		// if it's a route to a one hop peer, ignore
		if c.oneHopPeers.Contains(dst) {
			continue
		}
		baseRoute := c.lookup(from)
		// if we don't have a route to the peer that sent it, ignore
		if baseRoute == nil {
			continue
		}
		r2 := ConcatRoutes(baseRoute, r)
		if !c.routeTable.WouldAdd(r2) {
			continue
		}
		eg.Go(func() error {
			_, err := c.pinger.ping(ctx, from, r2.Path)
			if err != nil {
				return err
			}
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

func (c *crawler) lookup(target Addr) *Route {
	if c.oneHopPeers.Contains(target) {
		linkIndex := c.linkMap.ID2Int(target)
		return &Route{
			Dst:       target[:],
			Path:      Path{linkIndex},
			Timestamp: time.Now().Unix(),
		}
	}
	return c.routeTable.Get(target)
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
	for _, peerID := range c.oneHopPeers.ListPeers() {
		if err := c.crawlSingle(ctx, peerID); retErr == nil {
			retErr = err
		}
	}
	c.routeTable.ForEach(func(r *Route) bool {
		if r.Timestamp > lastRun {
			dst := idFromBytes(r.Dst)
			err := c.crawlSingle(ctx, dst)
			if retErr == nil {
				retErr = err
			}
		}
		return true
	})
	return retErr
}

func (c *crawler) crawlSingle(ctx context.Context, dst Addr) error {
	return c.sqf(ctx, dst, &QueryRoutes{
		Locus: c.routeTable.cache.Locus(),
		Nbits: uint32(c.routeTable.MustMatchBits()),
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
	defer c.routeTable.Delete(dst[:])
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
