package beaconnet

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/networks/neteng"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
)

type Router struct {
	log                        networks.Logger
	privateKey                 inet256.PrivateKey
	peers                      networks.PeerSet
	localID                    inet256.Addr
	peerStateTTL, beaconPeriod time.Duration

	lastCleanup time.Time
	lastBeacon  time.Time
	mu          sync.Mutex
	peerStates  map[inet256.Addr]peerState
	ourBeacon   *Beacon
}

func NewRouter(log networks.Logger) neteng.Router {
	return &Router{
		log:          log,
		peerStateTTL: defaultPeerStateTTL,
		beaconPeriod: defaultBeaconPeriod,

		peerStates: make(map[inet256.Addr]peerState),
	}
}

func (r *Router) Reset(privateKey inet256.PrivateKey, peers networks.PeerSet, getPublicKey neteng.PublicKeyFunc, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.privateKey = privateKey
	r.peers = peers
	r.localID = inet256.NewAddr(privateKey.Public())
	r.ourBeacon = newBeacon(privateKey, now)

	for k := range r.peerStates {
		delete(r.peerStates, k)
	}
}

func (r *Router) HandleAbove(dst inet256.Addr, data p2p.IOVec, send neteng.SendFunc) bool {
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
	r.send(send, next, dst, TypeData, data)
	return true
}

func (r *Router) HandleBelow(from inet256.Addr, data []byte, send neteng.SendFunc, deliver neteng.DeliverFunc, info neteng.InfoFunc) {
	if err := r.handleMessage(send, deliver, info, from, data); err != nil {
		r.log.Warn(err)
	}
}

func (r *Router) Heartbeat(now time.Time, send neteng.SendFunc) {
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

		r.broadcast(send, r.localID, TypeBeacon, body)
		r.lastBeacon = now
	}
}

func (r *Router) FindAddr(send neteng.SendFunc, info neteng.InfoFunc, prefix []byte, nbits int) {
	// we don't use send here because there is nothing to do to accelerate receiveing new peer information.
	addr, pubKey := func() (inet256.Addr, inet256.PublicKey) {
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
		info(addr, pubKey)
		return
	}
}

func (r *Router) LookupPublicKey(send neteng.SendFunc, info neteng.InfoFunc, target inet256.Addr) {
	ps := r.lookupPeerState(target)
	if ps != nil {
		info(target, ps.publicKey)
		return
	}
}

func (r *Router) handleMessage(send neteng.SendFunc, deliver neteng.DeliverFunc, info neteng.InfoFunc, from inet256.Addr, payload []byte) error {
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
		return r.handleBeacon(send, info, from, hdr, beacon)
	case TypeData:
		return r.handleData(send, deliver, from, hdr, body)
	default:
		return errors.Errorf("invalid message type %v", hdr.GetType())
	}
}

// handleBeacon handles updating the network state when a Beacon is received
func (r *Router) handleBeacon(send neteng.SendFunc, info neteng.InfoFunc, prev inet256.Addr, hdr Header, b Beacon) error {
	pubKey, err := verifyBeacon(b)
	if err != nil {
		return err
	}
	updated, created := r.updatePeerState(prev, pubKey, b.Counter)
	data := jsonMarshal(b)
	// a broadcast beacon that we haven't seen.
	if updated && hdr.GetDst() == broadcastAddr {
		r.broadcast(send, prev, TypeBeacon, data)
	}
	// a targetted beacon
	if hdr.GetDst() != broadcastAddr {
		ps := r.lookupPeerState(hdr.GetDst())
		if ps != nil {
			r.send(send, ps.next, hdr.GetDst(), TypeBeacon, p2p.IOVec{data})
		}
	}
	if created {
		// our first time hearing from this peer, respond immediately.
		r.mu.Lock()
		ourBeacon := r.ourBeacon
		r.mu.Unlock()
		body := jsonMarshal(ourBeacon)
		r.send(send, prev, hdr.GetSrc(), TypeBeacon, p2p.IOVec{body})
	}
	info(inet256.NewAddr(pubKey), pubKey)
	return nil
}

func (r *Router) handleData(send neteng.SendFunc, deliver neteng.DeliverFunc, prev inet256.Addr, hdr Header, body []byte) error {
	dst := hdr.GetDst()
	src := hdr.GetSrc()
	switch {
	// local
	case dst == r.localID:
		deliver(src, body)

	// neighbor
	case r.peers.Contains(dst):
		send(dst, p2p.IOVec{hdr[:], body})

	// forward
	default:
		ps := r.lookupPeerState(dst)
		if ps == nil {
			return fmt.Errorf("cannot forward. don't know next hop to %v", dst)
		}
		send(ps.next, p2p.IOVec{hdr[:], body})
	}
	return nil
}

func (r *Router) updatePeerState(next inet256.Addr, pubKey p2p.PublicKey, counter uint64) (updated, created bool) {
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

func (r *Router) broadcast(send neteng.SendFunc, exclude inet256.Addr, mtype uint8, body []byte) {
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

func (r *Router) send(send neteng.SendFunc, next, dst inet256.Addr, mtype uint8, body p2p.IOVec) {
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
	publicKey p2p.PublicKey
}

func (ps peerState) String() string {
	return fmt.Sprintf("{next: %v, counter: %d}", ps.next, ps.counter)
}
