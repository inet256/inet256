package beaconnet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	defaultBeaconPeriod = 1 * time.Second
	defaultPeerStateTTL = 30 * time.Second
	findAddrTimeout     = 10 * time.Second
)

type Network struct {
	privateKey                 p2p.PrivateKey
	log                        *logrus.Logger
	swarm                      inet256.Swarm
	peerSet                    inet256.PeerSet
	beaconPeriod, peerStateTTL time.Duration

	sg         netutil.ServiceGroup
	tellHub    *netutil.TellHub
	mu         sync.Mutex
	peerStates map[inet256.Addr]peerState
}

func Factory(params inet256.NetworkParams) inet256.Network {
	return New(params)
}

func New(params inet256.NetworkParams) inet256.Network {
	n := &Network{
		privateKey:   params.PrivateKey,
		log:          params.Logger,
		swarm:        params.Swarm,
		peerSet:      params.Peers,
		beaconPeriod: defaultBeaconPeriod,
		peerStateTTL: defaultPeerStateTTL,

		tellHub: netutil.NewTellHub(),

		peerStates: make(map[inet256.Addr]peerState),
	}
	n.sg.Go(n.broadcastBeaconLoop)
	n.sg.Go(n.cleanupLoop)
	n.sg.Go(n.recvLoops)
	return n
}

func (n *Network) Receive(ctx context.Context, fn func(inet256.Message)) error {
	return n.tellHub.Receive(ctx, fn)
}

func (n *Network) Tell(ctx context.Context, dst inet256.Addr, data []byte) error {
	if n.peerSet.Contains(dst) {
		return n.send(ctx, dst, dst, TypeData, data)
	}
	if _, err := n.FindAddr(ctx, dst[:], len(dst)*8); err != nil {
		return err
	}
	peerState := n.lookupPeerState(dst)
	if peerState == nil {
		return errors.Errorf("no state for peer %v", dst)
	}
	return n.send(ctx, peerState.next, dst, TypeData, data)
}

func (n *Network) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	if n.peerSet.Contains(target) {
		return n.swarm.LookupPublicKey(ctx, target)
	}
	if _, err := n.FindAddr(ctx, target[:], len(target)*8); err != nil {
		return nil, err
	}
	ps := n.lookupPeerState(target)
	if ps == nil {
		return nil, errors.Errorf("no state for peer %v", target)
	}
	return ps.publicKey, nil
}

func (n *Network) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	for _, id := range n.peerSet.ListPeers() {
		if inet256.HasPrefix(id[:], prefix, nbits) {
			return id, nil
		}
	}
	ctx, cf := context.WithTimeout(ctx, 10*time.Second)
	defer cf()
	var result inet256.Addr
	err := netutil.Retry(ctx, func() error {
		n.mu.Lock()
		defer n.mu.Unlock()
		for id := range n.peerStates {
			if inet256.HasPrefix(id[:], prefix, nbits) {
				result = id
				return nil
			}
		}
		return errors.Errorf("no state for peer matching prefix %x", prefix)
	}, netutil.WithPulseTrain(netutil.NewLinear(100*time.Millisecond)))
	return result, err
}

func (n *Network) LocalAddr() inet256.Addr {
	return inet256.NewAddr(n.privateKey.Public())
}

func (n *Network) PublicKey() inet256.PublicKey {
	return n.privateKey.Public()
}

func (n *Network) MTU(ctx context.Context, addr inet256.Addr) int {
	return inet256.TransportMTU - HeaderSize
}

func (n *Network) Bootstrap(ctx context.Context) error {
	return n.broadcastBeacon(ctx)
}

func (n *Network) Close() error {
	var el netutil.ErrList
	n.tellHub.CloseWithError(inet256.ErrClosed)
	el.Do(n.sg.Stop)
	el.Do(n.swarm.Close)
	return el.Err()
}

func (n *Network) DumpState() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	buf := bytes.NewBuffer(nil)
	fmt.Fprintln(buf, n.LocalAddr())
	fmt.Fprintf(buf, "PEER STATES len=%d\n", len(n.peerStates))
	fmt.Fprint(buf, n.peerStates)
	fmt.Fprintln(buf)
	return buf.String()
}

func (nwk *Network) recvLoop(ctx context.Context) error {
	var msg2 inet256.Message
	for {
		if err := nwk.swarm.Receive(ctx, func(msg inet256.Message) {
			msg2 = inet256.Message{
				Src:     msg.Src,
				Dst:     msg.Dst,
				Payload: append(msg2.Payload[:0], msg.Payload...),
			}
		}); err != nil {
			return err
		}
		if err := nwk.handleMessage(ctx, msg2); err != nil {
			nwk.log.Warnf("while handling message from %v: %v", msg2.Src, err)
		}
	}
}

func (nwk *Network) recvLoops(ctx context.Context) error {
	wp := netutil.WorkerPool{
		Fn: func(ctx context.Context) {
			err := nwk.recvLoop(ctx)
			if err != nil && !errors.Is(err, ctx.Err()) && !inet256.IsErrClosed(err) {
				nwk.log.Errorf("exiting recvLoop %v", err)
			}
		},
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		l := len(nwk.peerSet.ListPeers())
		wp.SetCount(l + 1)
		select {
		case <-ticker.C:
		case <-ctx.Done():
			wp.Stop()
			return ctx.Err()
		}
	}
}

func (n *Network) handleMessage(ctx context.Context, msg inet256.Message) error {
	hdr, body, err := ParseMessage(msg.Payload)
	if err != nil {
		return err
	}
	switch hdr.GetType() {
	case TypeBeacon:
		var beacon Beacon
		if err := json.Unmarshal(body, &beacon); err != nil {
			return err
		}
		return n.handleBeacon(ctx, msg.Src, hdr, beacon)
	case TypeData:
		return n.handleData(ctx, msg.Src, hdr, body)
	default:
		return errors.Errorf("invalid message type %v", hdr.GetType())
	}
}

// handleBeacon handles updating the network state when a Beacon is received
func (n *Network) handleBeacon(ctx context.Context, prev inet256.Addr, hdr Header, b Beacon) error {
	pubKey, err := verifyBeacon(b)
	if err != nil {
		return err
	}
	updated := n.updatePeerState(prev, pubKey, b.Counter)
	data := jsonMarshal(b)
	// a broadcast beacon that we haven't seen.
	if updated && hdr.GetDst() == broadcastAddr {
		if err := n.broadcast(ctx, prev, TypeBeacon, data); err != nil {
			return err
		}
	}
	// a targetted beacon
	if hdr.GetDst() != broadcastAddr {
		ps := n.lookupPeerState(hdr.GetDst())
		if ps != nil {
			if err := n.send(ctx, ps.next, hdr.GetDst(), TypeBeacon, data); err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Network) handleData(ctx context.Context, prev inet256.Addr, hdr Header, body []byte) error {
	dst := hdr.GetDst()
	src := hdr.GetSrc()
	switch {
	// local
	case dst == n.LocalAddr():
		return n.tellHub.Deliver(ctx, inet256.Message{
			Src:     src,
			Dst:     dst,
			Payload: body,
		})

	// neighbor
	case n.peerSet.Contains(dst):
		return n.swarm.Tell(ctx, dst, p2p.IOVec{hdr[:], body})

	// forward
	default:
		ps := n.lookupPeerState(dst)
		if ps == nil {
			return errors.Errorf("cannot forward. don't know next hop to %v", dst)
		}
		const timeout = 3 * time.Second
		ctx, cf := context.WithTimeout(context.Background(), timeout)
		defer cf()
		return n.swarm.Tell(ctx, ps.next, p2p.IOVec{hdr[:], body})
	}
}

func (n *Network) updatePeerState(next inet256.Addr, pubKey p2p.PublicKey, counter uint64) (updated bool) {
	peer := inet256.NewAddr(pubKey)
	n.mu.Lock()
	defer n.mu.Unlock()
	ps, exists := n.peerStates[peer]
	if !exists || counter > ps.counter {
		updated = true
		ps = peerState{
			counter:   counter,
			next:      next,
			publicKey: pubKey,
		}
	}
	n.peerStates[peer] = ps
	return updated
}

func (n *Network) lookupPeerState(dst inet256.Addr) *peerState {
	n.mu.Lock()
	defer n.mu.Unlock()
	peerState, exists := n.peerStates[dst]
	if !exists {
		return nil
	}
	return &peerState
}

func (n *Network) broadcastBeaconLoop(ctx context.Context) error {
	period := n.beaconPeriod
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := func() error {
				ctx, cf := context.WithTimeout(ctx, period/3)
				defer cf()
				return n.broadcastBeacon(ctx)
			}(); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// broadcastBeacon sends a beacon out to all peers.
func (n *Network) broadcastBeacon(ctx context.Context) error {
	now := time.Now()
	b := newBeacon(n.privateKey, now)
	body := jsonMarshal(b)
	return n.broadcast(ctx, n.LocalAddr(), TypeBeacon, body)
}

// sendBeacon sends a beacon to next.
func (n *Network) sendBeacon(ctx context.Context, next, dst inet256.Addr) error {
	now := time.Now()
	b := newBeacon(n.privateKey, now)
	body := jsonMarshal(b)
	return n.send(ctx, next, dst, TypeBeacon, body)
}

// broadcast sends a message with mtype and body to all peers
func (n *Network) broadcast(ctx context.Context, exclude inet256.Addr, mtype uint8, body []byte) error {
	hdr := Header{}
	hdr.SetSrc(n.LocalAddr())
	hdr.SetDst(broadcastAddr)
	hdr.SetType(mtype)
	eg := errgroup.Group{}
	for _, peerID := range n.peerSet.ListPeers() {
		if peerID == exclude {
			continue
		}
		peerID := peerID
		eg.Go(func() error {
			err := n.swarm.Tell(ctx, peerID, p2p.IOVec{hdr[:], body})
			if inet256.IsUnreachable(err) {
				err = nil
			}
			err = errors.Wrapf(err, "tell %v: ", peerID)
			return err
		})
	}
	return errors.Wrapf(eg.Wait(), "during broadcast")
}

// send sends a message with dst, mtype and body to peer next.
func (n *Network) send(ctx context.Context, next, dst inet256.Addr, mtype uint8, body []byte) error {
	hdr := Header{}
	hdr.SetSrc(n.LocalAddr())
	hdr.SetDst(dst)
	hdr.SetType(mtype)
	return n.swarm.Tell(ctx, next, p2p.IOVec{hdr[:], body})
}

func (n *Network) cleanupLoop(ctx context.Context) error {
	period := n.peerStateTTL
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cf := context.WithTimeout(ctx, period/3)
			defer cf()
			if err := n.cleanupOnce(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *Network) cleanupOnce(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := time.Now().UTC()
	nowCounter := uint64(now.UnixNano())
	for addr, ps := range n.peerStates {
		if time.Duration(nowCounter-ps.counter) > n.peerStateTTL {
			delete(n.peerStates, addr)
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
