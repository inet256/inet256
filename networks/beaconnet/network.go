package beaconnet

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Network struct {
	privateKey p2p.PrivateKey
	log        *logrus.Logger
	swarm      peerswarm.Swarm
	peerSet    inet256.PeerSet
	recvHub    *inet256.RecvHub
	cf         context.CancelFunc

	mu         sync.Mutex
	peerStates map[inet256.Addr]peerState
}

func Factory(params inet256.NetworkParams) inet256.Network {
	return New(params)
}

func New(params inet256.NetworkParams) inet256.Network {
	ctx, cf := context.WithCancel(context.Background())
	n := &Network{
		privateKey: params.PrivateKey,
		log:        params.Logger,
		swarm:      params.Swarm,
		peerSet:    params.Peers,
		recvHub:    inet256.NewRecvHub(),
		cf:         cf,

		peerStates: make(map[p2p.PeerID]peerState),
	}
	go n.broadcastLoop(ctx)
	go n.swarm.ServeTells(n.fromBelow)
	return n
}

func (n *Network) WaitReady(ctx context.Context) error {
	return n.broadcastOnce(ctx)
}

func (n *Network) Recv(fn inet256.RecvFunc) error {
	return n.recvHub.Recv(fn)
}

func (n *Network) Tell(ctx context.Context, dst inet256.Addr, data []byte) error {
	peerState := n.lookupPeerState(dst)
	if peerState == nil {
		return errors.Errorf("no state for peer %v", dst)
	}
	msg := Message{
		Data: &DataMessage{
			Src:     p2p.NewPeerID(n.privateKey.Public()),
			Dst:     dst,
			Payload: data,
		},
	}
	return n.swarm.TellPeer(ctx, peerState.next, p2p.IOVec{jsonMarshal(msg)})
}

func (n *Network) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	if n.peerSet.Contains(target) {
		return n.swarm.LookupPublicKey(ctx, target)
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
	n.mu.Lock()
	defer n.mu.Unlock()
	for id := range n.peerStates {
		if inet256.HasPrefix(id[:], prefix, nbits) {
			return id, nil
		}
	}
	return inet256.Addr{}, nil
}

func (n *Network) LocalAddr() inet256.Addr {
	return p2p.NewPeerID(n.privateKey.Public())
}

func (n *Network) MTU(ctx context.Context, addr inet256.Addr) int {
	return inet256.TransportMTU - 128
}

func (n *Network) Close() error {
	err := n.swarm.Close()
	n.recvHub.CloseWithError(err)
	return err
}

func (n *Network) fromBelow(msg *p2p.Message) {
	src := msg.Src.(p2p.PeerID)
	if err := func() error {
		var m Message
		if err := json.Unmarshal(msg.Payload, &m); err != nil {
			return err
		}
		switch {
		case m.Beacon != nil:
			return n.handleBeacon(src, m.Beacon)
		case m.Data != nil:
			return n.handleData(src, m.Data)
		}
		return nil
	}(); err != nil {
		n.log.Errorf("invalid message from %v", msg.Src)
	}
}

func (n *Network) handleBeacon(prev inet256.Addr, b *Beacon) error {
	pubKey, err := p2p.ParsePublicKey(b.PublicKey)
	if err != nil {
		return err
	}
	peer := p2p.NewPeerID(pubKey)
	n.mu.Lock()
	var updated bool
	ps, exists := n.peerStates[peer]
	if !exists || b.Counter > ps.counter {
		updated = true
		ps = peerState{
			counter:   b.Counter,
			next:      prev,
			publicKey: pubKey,
		}
	}
	n.peerStates[peer] = ps
	n.mu.Unlock()
	if updated {
		ctx := context.Background()
		eg := errgroup.Group{}
		for _, id := range n.peerSet.ListPeers() {
			if id == prev {
				continue
			}
			id := id
			data := jsonMarshal(Message{
				Beacon: b,
			})
			eg.Go(func() error {
				return n.swarm.TellPeer(ctx, id, p2p.IOVec{data})
			})
		}
		if err := eg.Wait(); err != nil {
			return err
		}
		return n.broadcastOnce(ctx)
	}
	if !exists {
		return n.broadcastOnce(context.Background())
	}
	return nil
}

func (n *Network) handleData(prev inet256.Addr, dm *DataMessage) error {
	if dm.Dst == n.LocalAddr() {
		n.recvHub.Deliver(dm.Src, dm.Dst, dm.Payload)
		return nil
	}
	ps := n.lookupPeerState(dm.Dst)
	if ps == nil {
		return errors.Errorf("cannot forward to %v", dm.Dst)
	}
	const timeout = 3 * time.Second
	ctx, cf := context.WithTimeout(context.Background(), timeout)
	defer cf()
	data := jsonMarshal(Message{Data: dm})
	return n.swarm.TellPeer(ctx, ps.next, p2p.IOVec{data})
}

func (n *Network) lookupPeerState(dst inet256.Addr) *peerState {
	n.mu.Lock()
	defer n.mu.Unlock()
	peerState, exists := n.peerStates[dst]
	if !exists {
		return nil
	}
	const stateStaleTimeout = time.Minute
	now := time.Now().UTC()
	if uint64(now.UnixNano()-stateStaleTimeout.Nanoseconds()) < peerState.counter {
		delete(n.peerStates, dst)
		return nil
	}
	return &peerState
}

func (n *Network) broadcastLoop(ctx context.Context) error {
	const period = 3 * time.Second
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cf := context.WithTimeout(ctx, period/3)
			defer cf()
			if err := n.broadcastOnce(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

const sigPurpose = "inet256/beaconnet/beacon"

func (n *Network) broadcastOnce(ctx context.Context) error {
	now := time.Now()
	msg := Message{
		Beacon: newBeacon(n.privateKey, now),
	}
	data := jsonMarshal(msg)
	eg := errgroup.Group{}
	for _, peerID := range n.peerSet.ListPeers() {
		peerID := peerID
		eg.Go(func() error {
			return n.swarm.TellPeer(ctx, peerID, p2p.IOVec{data})
		})
	}
	return eg.Wait()
}

func newBeacon(privateKey p2p.PrivateKey, now time.Time) *Beacon {
	now = now.UTC()
	counter := now.UnixNano()
	counterBytes := [8]byte{}
	binary.BigEndian.PutUint64(counterBytes[:], uint64(counter))
	sig, err := p2p.Sign(nil, privateKey, sigPurpose, counterBytes[:])
	if err != nil {
		panic(err)
	}
	return &Beacon{
		PublicKey: p2p.MarshalPublicKey(privateKey.Public()),
		Counter:   uint64(now.Unix()),
		Sig:       sig,
	}
}

func jsonMarshal(x interface{}) []byte {
	data, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	return data
}

type peerState struct {
	next      inet256.Addr
	counter   uint64
	publicKey p2p.PublicKey
}

type Beacon struct {
	PublicKey []byte `json:"public_key"`
	Counter   uint64 `json:"counter"`
	Sig       []byte `json:"sig"`
}

type DataMessage struct {
	Src     inet256.Addr `json:"src"`
	Dst     inet256.Addr `json:"dst"`
	Payload []byte       `json:"payload"`
}

type Message struct {
	Beacon *Beacon      `json:"beacon,omitempty"`
	Data   *DataMessage `json:"data,omitempty"`
}
