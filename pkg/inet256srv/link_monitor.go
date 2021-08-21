package inet256srv

import (
	"context"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// linkMonitor sends out heartbeats to all addresses in the peer store
// and keeps track of where heartbeats are coming from.
type linkMonitor struct {
	x           p2p.SecureSwarm
	peerStore   PeerStore
	expireAfter time.Duration
	log         *Logger
	sg          netutil.ServiceGroup

	mu        sync.RWMutex
	sightings map[inet256.Addr]map[string]time.Time
	addrs     map[string]p2p.Addr
}

func newLinkMonitor(x p2p.SecureSwarm, peerStore PeerStore, log *Logger) *linkMonitor {
	const expireAfter = 30 * time.Second
	lm := &linkMonitor{
		x:           x,
		peerStore:   peerStore,
		log:         log,
		expireAfter: expireAfter,
		sightings:   make(map[inet256.Addr]map[string]time.Time),
		addrs:       make(map[string]p2p.Addr),
	}
	lm.sg.Go(lm.recvLoop)
	lm.sg.Go(lm.heartbeatLoop)
	lm.sg.Go(lm.cleanupLoop)
	return lm
}

func (lm *linkMonitor) recvLoop(ctx context.Context) error {
	buf := make([]byte, lm.x.MaxIncomingSize())
	for {
		var src, dst p2p.Addr
		_, err := lm.x.Receive(ctx, &src, &dst, buf)
		if err != nil {
			return err
		}
		pubKey, err := lm.x.LookupPublicKey(ctx, src)
		if err != nil {
			lm.log.Error("in linkMonitor.recvLoop: ", err)
			continue
		}
		lm.Mark(inet256.NewAddr(pubKey), src, time.Now())
	}
}

func (lm *linkMonitor) heartbeatLoop(ctx context.Context) error {
	const period = 5 * time.Second
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		func() {
			ctx, cf := context.WithTimeout(ctx, period*2/3)
			defer cf()
			if err := lm.heartbeat(ctx); err != nil {
				lm.log.Error("during heartbeat: ", err)
			}
		}()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (lm *linkMonitor) heartbeat(ctx context.Context) error {
	data := []byte(time.Now().String())
	eg := errgroup.Group{}
	for _, id := range lm.peerStore.ListPeers() {
		id := id
		for _, addr := range lm.peerStore.ListAddrs(id) {
			addr := addr
			eg.Go(func() error {
				return lm.x.Tell(ctx, addr, p2p.IOVec{data})
			})
		}
	}
	return eg.Wait()
}

func (lm *linkMonitor) Mark(id inet256.Addr, a p2p.Addr, t time.Time) {
	data, _ := a.MarshalText()
	lm.mu.Lock()
	defer lm.mu.Unlock()
	m := lm.sightings[id]
	if m == nil {
		m = make(map[string]time.Time)
		lm.sightings[id] = m
	}
	current := m[string(data)]
	if t.After(current) {
		m[string(data)] = t
		lm.addrs[string(data)] = a
	}
}

func (lm *linkMonitor) PickAddr(id inet256.Addr) (p2p.Addr, error) {
	if !lm.peerStore.Contains(id) {
		lm.log.Error("peers in store:", lm.peerStore.ListPeers())
		return nil, errors.Errorf("cannot pick address for peer not in store %v", id)
	}
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	// check for a known good address.
	m := lm.sightings[id]
	if m != nil {
		var latestAddr p2p.Addr
		var latestTime time.Time
		for addr, seenAt := range m {
			if seenAt.After(latestTime) {
				var exists bool
				latestTime = seenAt
				latestAddr, exists = lm.addrs[addr]
				if !exists {
					panic("missing address")
				}
			}
		}
		if latestAddr != nil {
			return latestAddr, nil
		}
	}
	// pick a random address from the store.
	addrs := lm.peerStore.ListAddrs(id)
	if len(addrs) == 0 {
		return nil, errors.Errorf("no transport addresses for peer %v", id)
	}
	addr := addrs[mrand.Intn(len(addrs))]
	return addr, nil
}

func (lm *linkMonitor) Close() error {
	lm.sg.Stop()
	return lm.x.Close()
}

func (lm *linkMonitor) LastSeen(id inet256.Addr) map[string]time.Time {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return copyLastSeen(lm.sightings[id])
}

func (lm *linkMonitor) cleanupLoop(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		now := time.Now()
		lm.cleanupOnce(ctx, now)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (lm *linkMonitor) cleanupOnce(ctx context.Context, now time.Time) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	for id, addrs := range lm.sightings {
		for addr, lastSeen := range addrs {
			if now.Sub(lastSeen) > lm.expireAfter {
				delete(addrs, addr)
			}
		}
		if len(lm.sightings) == 0 {
			delete(lm.sightings, id)
		}
	}
}

func copyLastSeen(x map[string]time.Time) map[string]time.Time {
	y := make(map[string]time.Time, len(x))
	for k, v := range x {
		y[k] = v
	}
	return y
}
