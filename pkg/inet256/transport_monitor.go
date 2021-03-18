package inet256

import (
	"context"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// transportMonitor sends out heartbeats to all addresses in the peer store
// and keeps track of where heartbeats are coming from.
type transportMonitor struct {
	x         p2p.SecureSwarm
	peerStore PeerStore
	cf        context.CancelFunc

	mu        sync.RWMutex
	sightings map[p2p.PeerID]map[string]time.Time
	addrs     map[string]p2p.Addr
}

func newTransportMonitor(x p2p.SecureSwarm, peerStore PeerStore) *transportMonitor {
	ctx, cf := context.WithCancel(context.Background())
	tm := &transportMonitor{
		x:         x,
		peerStore: peerStore,
		cf:        cf,
		sightings: make(map[p2p.PeerID]map[string]time.Time),
	}
	go tm.heartbeatLoop(ctx)
	go tm.cleanupLoop(ctx)
	go x.ServeTells(tm.handleTell)
	return tm
}

func (tm *transportMonitor) handleTell(msg *p2p.Message) {
	srcKey := p2p.LookupPublicKeyInHandler(tm.x, msg.Src)
	tm.Mark(p2p.NewPeerID(srcKey), msg.Src, time.Now())
}

func (tm *transportMonitor) heartbeatLoop(ctx context.Context) {
	const period = 5 * time.Second
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		func() {
			ctx, cf := context.WithTimeout(ctx, period*2/3)
			defer cf()
			if err := tm.heartbeat(ctx); err != nil {
				logrus.Error("during heartbeat: ", err)
			}
		}()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (tm *transportMonitor) heartbeat(ctx context.Context) error {
	eg := errgroup.Group{}
	for _, id := range tm.peerStore.ListPeers() {
		id := id
		for _, addrStr := range tm.peerStore.ListAddrs(id) {
			addrStr := addrStr
			eg.Go(func() error {
				addr, err := tm.x.ParseAddr([]byte(addrStr))
				if err != nil {
					return err
				}
				return tm.x.Tell(ctx, addr, p2p.IOVec{})
			})
		}
	}
	return eg.Wait()
}

func (tm *transportMonitor) Mark(id p2p.PeerID, a p2p.Addr, t time.Time) {
	data, _ := a.MarshalText()
	tm.mu.Lock()
	defer tm.mu.Unlock()
	m := tm.sightings[id]
	if m == nil {
		tm.sightings[id] = make(map[string]time.Time)
		m = tm.sightings[id]
	}
	current := m[string(data)]
	if t.After(current) {
		m[string(data)] = t
	}
}

func (tm *transportMonitor) PickAddr(id p2p.PeerID) (p2p.Addr, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	// check for a known good address.
	m := tm.sightings[id]
	if m != nil {
		var latestAddr p2p.Addr
		var latestTime time.Time
		for addr, seenAt := range m {
			if seenAt.After(latestTime) {
				latestTime = seenAt
				latestAddr = tm.addrs[addr]
			}
		}
		if latestAddr != nil {
			return latestAddr, nil
		}
	}
	// pick a random address from the store.
	addrs := tm.peerStore.ListAddrs(id)
	if len(addrs) == 0 {
		return nil, errors.Errorf("no transport addresses for peer %v", id)
	}
	addr := addrs[mrand.Intn(len(addrs))]
	return tm.x.ParseAddr([]byte(addr))
}

func (tm *transportMonitor) Close() error {
	tm.cf()
	return tm.x.Close()
}

func (tm *transportMonitor) LastSeen(id p2p.PeerID) map[string]time.Time {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return copyLastSeen(tm.sightings[id])
}

func (tm *transportMonitor) cleanupLoop(ctx context.Context) {
	const expireAfter = 10 * time.Minute
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		func() {
			now := time.Now()
			tm.mu.Lock()
			defer tm.mu.Unlock()
			for _, addrs := range tm.sightings {
				for addr, lastSeen := range addrs {
					if now.Sub(lastSeen) > expireAfter {
						delete(addrs, addr)
					}
				}
			}
		}()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
