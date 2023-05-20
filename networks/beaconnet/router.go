package beaconnet

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/pkg/errors"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/maybe"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/mesh256/routers"
)

type Router struct {
	privateKey                 inet256.PrivateKey
	peers                      mesh256.PeerSet
	localID                    inet256.Addr
	peerStateTTL, beaconPeriod time.Duration

	lastCleanup time.Time
	lastBeacon  time.Time
	mu          sync.Mutex
	peerStates  map[inet256.Addr]peerState
	ourBeacon   *Beacon
}

func NewRouter() routers.Router {
	return &Router{
		peerStateTTL: defaultPeerStateTTL,
		beaconPeriod: defaultBeaconPeriod,

		peerStates: make(map[inet256.Addr]peerState),
	}
}

func (r *Router) Reset(p routers.RouterParams) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.privateKey = p.PrivateKey
	r.peers = p.Peers
	r.localID = inet256.NewAddr(p.PrivateKey.Public())
	r.ourBeacon = newBeacon(p.PrivateKey, p.Now)

	for k := range r.peerStates {
		delete(r.peerStates, k)
	}
}

func (r *Router) HandleAbove(ctx *routers.AboveContext, dst inet256.Addr, data p2p.IOVec) bool {
	var next inet256.Addr
	if r.peers.Contains(dst) {
		next = dst
	} else {
		ps := r.lookupPeerState(dst)
		if ps == nil {
			return false
		}
		next = ps.next
	}
	r.send(ctx.Send, next, dst, TypeData, data)
	return true
}

func (r *Router) HandleBelow(ctx *routers.BelowContext, from inet256.Addr, data []byte) {
	if err := r.handleMessage(ctx, from, data); err != nil {
		logctx.Warnln(ctx, "handle below: ", err)
	}
}

func (r *Router) Heartbeat(ctx *routers.AboveContext, now time.Time) {
	if r.lastCleanup.Sub(now) >= r.peerStateTTL {
		r.cleanupOnce(now)
		r.lastCleanup = now
	}
	if now.Sub(r.lastBeacon) >= r.beaconPeriod {
		b := newBeacon(r.privateKey, now)
		r.mu.Lock()
		r.ourBeacon = b
		r.mu.Unlock()
		body := jsonMarshal(b)

		r.broadcast(ctx.Send, r.localID, TypeBeacon, body)
		r.lastBeacon = now
	}
}

func (r *Router) FindAddr(ctx *routers.AboveContext, prefix []byte, nbits int) maybe.Maybe[inet256.Addr] {
	// we don't use send here because there is nothing to do to accelerate receiveing new peer information.
	addr, _ := func() (inet256.Addr, inet256.PublicKey) {
		r.mu.Lock()
		defer r.mu.Unlock()
		for id, ps := range r.peerStates {
			if inet256.HasPrefix(id[:], prefix, nbits) {
				return id, ps.publicKey
			}
		}
		return inet256.Addr{}, nil
	}()
	if !addr.IsZero() {
		return maybe.Just(addr)
	}
	return maybe.Nothing[inet256.Addr]()
}

func (r *Router) LookupPublicKey(ctx *routers.AboveContext, target inet256.Addr) maybe.Maybe[inet256.PublicKey] {
	ps := r.lookupPeerState(target)
	if ps != nil {
		return maybe.Just(ps.publicKey)
	}
	return maybe.Nothing[inet256.PublicKey]()
}

func (r *Router) MTU(below int) int {
	return below - HeaderSize
}

func (r *Router) handleMessage(ctx *routers.BelowContext, from inet256.Addr, payload []byte) error {
	hdr, body, err := ParseMessage(payload)
	if err != nil {
		return err
	}
	switch hdr.GetType() {
	case TypeBeacon:
		var beacon Beacon
		if err := json.Unmarshal(body, &beacon); err != nil {
			return err
		}
		return r.handleBeacon(ctx, from, hdr, beacon)
	case TypeData:
		return r.handleData(ctx, from, hdr, body)
	default:
		return errors.Errorf("invalid message type %v", hdr.GetType())
	}
}

// handleBeacon handles updating the network state when a Beacon is received
func (r *Router) handleBeacon(ctx *routers.BelowContext, prev inet256.Addr, hdr Header, b Beacon) error {
	pubKey, err := verifyBeacon(b)
	if err != nil {
		return err
	}
	updated, created := r.updatePeerState(prev, pubKey, b.Counter)
	data := jsonMarshal(b)
	// a broadcast beacon that we haven't seen.
	if updated && hdr.GetDst() == broadcastAddr {
		r.broadcast(ctx.Send, prev, TypeBeacon, data)
	}
	// a targetted beacon
	if hdr.GetDst() != broadcastAddr {
		ps := r.lookupPeerState(hdr.GetDst())
		if ps != nil {
			r.send(ctx.Send, ps.next, hdr.GetDst(), TypeBeacon, p2p.IOVec{data})
		}
	}
	if created {
		// our first time hearing from this peer, respond immediately.
		r.mu.Lock()
		ourBeacon := r.ourBeacon
		r.mu.Unlock()
		body := jsonMarshal(ourBeacon)
		r.send(ctx.Send, prev, hdr.GetSrc(), TypeBeacon, p2p.IOVec{body})
	}
	addr := inet256.NewAddr(pubKey)
	ctx.OnAddr(addr)
	ctx.OnPublicKey(addr, pubKey)
	return nil
}

func (r *Router) handleData(ctx *routers.BelowContext, prev inet256.Addr, hdr Header, body []byte) error {
	dst := hdr.GetDst()
	src := hdr.GetSrc()
	switch {
	// local
	case dst == r.localID:
		ctx.OnData(src, body)

	// neighbor
	case r.peers.Contains(dst):
		ctx.Send(dst, p2p.IOVec{hdr[:], body})

	// forward
	default:
		ps := r.lookupPeerState(dst)
		if ps == nil {
			return fmt.Errorf("cannot forward. don't know next hop to %v", dst)
		}
		ctx.Send(ps.next, p2p.IOVec{hdr[:], body})
	}
	return nil
}

func (r *Router) updatePeerState(next inet256.Addr, pubKey inet256.PublicKey, counter uint64) (updated, created bool) {
	peer := inet256.NewAddr(pubKey)
	r.mu.Lock()
	defer r.mu.Unlock()
	ps, exists := r.peerStates[peer]
	if !exists || counter > ps.counter {
		updated = true
		ps = peerState{
			counter:   counter,
			next:      next,
			publicKey: pubKey,
		}
	}
	r.peerStates[peer] = ps
	return updated, !exists
}

func (r *Router) lookupPeerState(dst inet256.Addr) *peerState {
	r.mu.Lock()
	defer r.mu.Unlock()
	peerState, exists := r.peerStates[dst]
	if !exists {
		return nil
	}
	return &peerState
}

func (r *Router) broadcast(send routers.SendFunc, exclude inet256.Addr, mtype uint8, body []byte) {
	hdr := Header{}
	hdr.SetSrc(r.localID)
	hdr.SetDst(broadcastAddr)
	hdr.SetType(mtype)
	for _, peerID := range r.peers.ListPeers() {
		if peerID == exclude {
			continue
		}
		send(peerID, p2p.IOVec{hdr[:], body})
	}
}

func (r *Router) send(send routers.SendFunc, next, dst inet256.Addr, mtype uint8, body p2p.IOVec) {
	hdr := Header{}
	hdr.SetSrc(r.localID)
	hdr.SetDst(dst)
	hdr.SetType(mtype)
	send(next, append(p2p.IOVec{hdr[:]}, body...))
}

func (r *Router) cleanupOnce(now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	nowCounter := uint64(now.UnixNano())
	for addr, ps := range r.peerStates {
		if time.Duration(nowCounter-ps.counter) > r.peerStateTTL {
			delete(r.peerStates, addr)
		}
	}
	return nil
}

type peerState struct {
	next      inet256.Addr
	counter   uint64
	publicKey inet256.PublicKey
}

func (ps peerState) String() string {
	return fmt.Sprintf("{next: %v, counter: %d}", ps.next, ps.counter)
}
