package mesh256

import (
	"context"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/inet256/inet256/pkg/peers"
)

const expireAfter = 30 * time.Second

// linkMonitor sends out heartbeats to all addresses in the peer store
// and keeps track of where heartbeats are coming from.
type linkMonitor[A p2p.Addr, Pub any] struct {
	x           p2p.SecureSwarm[A, Pub]
	peerStore   peers.Store[A]
	expireAfter time.Duration
	convert     func(Pub) (inet256.Addr, error)
	sg          netutil.ServiceGroup

	mu        sync.RWMutex
	sightings map[inet256.Addr]map[string]time.Time
	addrs     map[string]A
}

// newLinkMonitor monitors all the peers in peerStore by sending messages over x.
// convert is called to convert a Pub to an inet256.Addr
func newLinkMonitor[A p2p.Addr, Pub any](bgCtx context.Context, x p2p.SecureSwarm[A, Pub], peerStore peers.Store[A], convert func(Pub) (inet256.Addr, error)) *linkMonitor[A, Pub] {
	lm := &linkMonitor[A, Pub]{
		x:           x,
		peerStore:   peerStore,
		expireAfter: expireAfter,
		convert:     convert,

		sightings: make(map[inet256.Addr]map[string]time.Time),
		addrs:     make(map[string]A),
	}
	lm.sg.Background = bgCtx
	lm.sg.Go(lm.recvLoop)
	lm.sg.Go(lm.heartbeatLoop)
	lm.sg.Go(lm.cleanupLoop)
	return lm
}

func (lm *linkMonitor[A, Pub]) recvLoop(ctx context.Context) error {
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
		addr, err := lm.convert(pubKey)
		if err != nil {
			logctx.Errorf(ctx, "linkMonitor: converting public key %v", err)
			continue
		}
		lm.Mark(addr, msg.Src, time.Now())
	}
}

func (lm *linkMonitor[A, Pub]) heartbeatLoop(ctx context.Context) error {
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

func (lm *linkMonitor[A, Pub]) heartbeat(ctx context.Context) error {
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

func (lm *linkMonitor[A, Pub]) Mark(id inet256.Addr, a A, t time.Time) {
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

func (lm *linkMonitor[A, Pub]) PickAddr(ctx context.Context, id inet256.Addr) (*A, error) {
	if !lm.peerStore.Contains(id) {
		logctx.Errorln(ctx, "peers in store:", lm.peerStore.ListPeers())
		return nil, errors.Errorf("cannot pick address for peer not in store %v", id)
	}
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	// check for a known good address.
	m := lm.sightings[id]
	if m != nil {
		var latestAddr *A
		var latestTime time.Time
		for addr, seenAt := range m {
			if seenAt.After(latestTime) {
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
	addrs := lm.peerStore.ListAddrs(id)
	if len(addrs) == 0 {
		return nil, inet256.ErrAddrUnreachable{Addr: id}
	}
	addr := addrs[mrand.Intn(len(addrs))]
	return &addr, nil
}

func (lm *linkMonitor[A, Pub]) Close() error {
	var el netutil.ErrList
	el.Add(lm.sg.Stop())
	el.Add(lm.x.Close())
	return el.Err()
}

func (lm *linkMonitor[A, Pub]) LastSeen(id inet256.Addr) map[string]time.Time {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return copyLastSeen(lm.sightings[id])
}

func (lm *linkMonitor[A, Pub]) cleanupLoop(ctx context.Context) error {
	ticker := time.NewTicker(lm.expireAfter / 2)
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

func (lm *linkMonitor[A, Pub]) cleanupOnce(ctx context.Context, now time.Time) {
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
