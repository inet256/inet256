package multihoming

import (
	"context"
	"fmt"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-tai64"
	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"

	"github.com/inet256/inet256/internal/netutil"
)

const expireAfter = 30 * time.Second

// linkMonitor sends out heartbeats to all addresses in the peer store
// and keeps track of where heartbeats are coming from.
type linkMonitor[A p2p.Addr, Pub any, K comparable] struct {
	x           p2p.SecureSwarm[A, Pub]
	peers       PeerStore[K, A]
	expireAfter time.Duration
	groupBy     func(Pub) (K, error)
	sg          netutil.ServiceGroup

	mu        sync.RWMutex
	sightings map[K]map[string]tai64.TAI64N
	addrs     map[string]A
}

// newLinkMonitor monitors all the peers in peerStore by sending messages over x.
// convert is called to convert a Pub to an inet256.Addr
func newLinkMonitor[A p2p.Addr, Pub any, K comparable](bgCtx context.Context, x p2p.SecureSwarm[A, Pub], peers PeerStore[K, A], groupBy func(Pub) (K, error)) *linkMonitor[A, Pub, K] {
	lm := &linkMonitor[A, Pub, K]{
		x:           x,
		peers:       peers,
		expireAfter: expireAfter,
		groupBy:     groupBy,

		sightings: make(map[K]map[string]tai64.TAI64N),
		addrs:     make(map[string]A),
	}
	lm.sg.Background = bgCtx
	lm.sg.Go(lm.recvLoop)
	lm.sg.Go(lm.heartbeatLoop)
	lm.sg.Go(lm.cleanupLoop)
	return lm
}

func (lm *linkMonitor[A, Pub, K]) recvLoop(ctx context.Context) error {
	var msg p2p.Message[A]
	for {
		if err := p2p.Receive[A](ctx, lm.x, &msg); err != nil {
			return err
		}
		pubKey, err := lm.x.LookupPublicKey(ctx, msg.Src)
		if err != nil {
			logctx.Errorln(ctx, "in linkMonitor.recvLoop: ", err)
			continue
		}
		k, err := lm.groupBy(pubKey)
		if err != nil {
			logctx.Errorf(ctx, "linkMonitor: converting public key %v", err)
			continue
		}
		lm.Mark(k, msg.Src, tai64.Now())
	}
}

func (lm *linkMonitor[A, Pub, K]) heartbeatLoop(ctx context.Context) error {
	const period = 5 * time.Second
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		func() {
			ctx, cf := context.WithTimeout(ctx, period*2/3)
			defer cf()
			if err := lm.heartbeat(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
				logctx.Errorln(ctx, "during heartbeat: ", err)
			}
		}()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (lm *linkMonitor[A, Pub, K]) heartbeat(ctx context.Context) error {
	now := tai64.Now()
	data := now.Marshal()
	eg := errgroup.Group{}
	for _, id := range lm.peers.ListPeers() {
		id := id
		for _, addr := range lm.peers.ListAddrs(id) {
			addr := addr
			eg.Go(func() error {
				return lm.x.Tell(ctx, addr, p2p.IOVec{data})
			})
		}
	}
	return eg.Wait()
}

func (lm *linkMonitor[A, Pub, K]) Mark(k K, a A, t tai64.TAI64N) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	m := lm.sightings[k]
	if m == nil {
		m = make(map[string]tai64.TAI64N)
		lm.sightings[k] = m
	}
	ka := keyForAddr(a)
	current := m[ka]
	if t.After(current) {
		m[ka] = t
		lm.addrs[ka] = a
	}
}

func (lm *linkMonitor[A, Pub, K]) PickAddr(ctx context.Context, k K) (*A, error) {
	if !lm.peers.Contains(k) {
		logctx.Errorln(ctx, "peers in store:", lm.peers.ListPeers())
		return nil, errors.Errorf("cannot pick address for peer not in store %v", k)
	}
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	// check for a known good address.
	m := lm.sightings[k]
	if m != nil {
		var latestAddr *A
		var latestTime tai64.TAI64N
		for addr, seenAt := range m {
			if (latestTime == tai64.TAI64N{}) || seenAt.After(latestTime) {
				var exists bool
				latestTime = seenAt
				a, exists := lm.addrs[addr]
				if !exists {
					panic("missing address")
				}
				latestAddr = &a
			}
		}
		if latestAddr != nil {
			return latestAddr, nil
		}
	}
	// pick a random address from the store.
	addrs := lm.peers.ListAddrs(k)
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no good addresses for %v", k)
	}
	addr := addrs[mrand.Intn(len(addrs))]
	return &addr, nil
}

func (lm *linkMonitor[A, Pub, K]) Close() error {
	var el netutil.ErrList
	el.Add(lm.sg.Stop())
	el.Add(lm.x.Close())
	return el.Err()
}

func (lm *linkMonitor[A, Pub, K]) LastSeen(k K) map[string]tai64.TAI64N {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return maps.Clone(lm.sightings[k])
}

func (lm *linkMonitor[A, Pub, K]) cleanupLoop(ctx context.Context) error {
	ticker := time.NewTicker(lm.expireAfter / 2)
	defer ticker.Stop()
	now := tai64.Now()
	for {
		lm.cleanupOnce(ctx, now)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ticker.C:
			now = tai64.FromGoTime(t)
		}
	}
}

func (lm *linkMonitor[A, Pub, K]) cleanupOnce(ctx context.Context, now tai64.TAI64N) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	for id, addrs := range lm.sightings {
		for addr, lastSeen := range addrs {
			if tai64DeltaGt(now, lastSeen, lm.expireAfter) {
				delete(addrs, addr)
			}
		}
		if len(lm.sightings) == 0 {
			delete(lm.sightings, id)
		}
	}
}

func tai64DeltaGt(a, b tai64.TAI64N, d time.Duration) bool {
	return a.GoTime().Sub(b.GoTime()) > d
}

func keyForAddr(x p2p.Addr) string {
	data, _ := x.MarshalText()
	return string(data)
}
