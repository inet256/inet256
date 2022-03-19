package floodnet

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"golang.org/x/sync/errgroup"
)

func Factory(params networks.Params) networks.Network {
	return New(params.PrivateKey, params.Swarm, params.Peers, params.Logger)
}

const maxHops = 10

type Addr = inet256.Addr

type Network struct {
	localAddr  inet256.Addr
	privateKey inet256.PrivateKey
	onehop     networks.PeerSet
	swarm      networks.Swarm
	log        *networks.Logger

	mu    sync.RWMutex
	peers map[Addr]inet256.PublicKey

	recvHub *netutil.TellHub
}

func New(privateKey inet256.PrivateKey, ps networks.Swarm, onehop networks.PeerSet, log *networks.Logger) *Network {
	n := &Network{
		localAddr:  inet256.NewAddr(privateKey.Public()),
		privateKey: privateKey,
		onehop:     onehop,
		swarm:      ps,
		log:        log,

		peers:   make(map[Addr]inet256.PublicKey),
		recvHub: netutil.NewTellHub(),
	}
	go func() {
		if err := n.recvLoop(context.Background()); err != nil {
			n.log.Error("exiting recvLoop with", err)
		}
	}()
	return n
}

func (n *Network) Tell(ctx context.Context, dst Addr, data p2p.IOVec) error {
	msg := newMessage(n.privateKey, dst, p2p.VecBytes(nil, data), maxHops, modeData)
	msgData, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return n.broadcast(ctx, msgData, n.localAddr)
}

func (n *Network) Receive(ctx context.Context, fn func(p2p.Message[inet256.Addr])) error {
	return n.recvHub.Receive(ctx, fn)
}

func (n *Network) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	if err := n.solicit(ctx); err != nil {
		return Addr{}, err
	}
	var ret Addr
	err := n.poll(ctx, func() error {
		n.mu.RLock()
		defer n.mu.RUnlock()
		for id := range n.peers {
			if inet256.HasPrefix(id[:], prefix, nbits) {
				ret = id
				return nil
			}
		}
		return inet256.ErrNoAddrWithPrefix
	})
	return ret, err
}

func (n *Network) LookupPublicKey(ctx context.Context, target Addr) (inet256.PublicKey, error) {
	if err := n.solicit(ctx); err != nil {
		return nil, err
	}
	var ret inet256.PublicKey
	err := n.poll(ctx, func() error {
		n.mu.RLock()
		defer n.mu.RUnlock()
		for id, pubKey := range n.peers {
			if id == target {
				ret = pubKey
			}
		}
		return inet256.ErrPublicKeyNotFound
	})
	return ret, err
}

func (n *Network) LocalAddr() Addr {
	return n.localAddr
}

func (n *Network) PublicKey() inet256.PublicKey {
	return n.privateKey.Public()
}

func (n *Network) Bootstrap(ctx context.Context) error {
	return n.solicit(ctx)
}

func (n *Network) MTU(ctx context.Context, target Addr) int {
	return inet256.MinMTU
}

func (n *Network) Close() error {
	return n.swarm.Close()
}

func (n *Network) poll(ctx context.Context, fn func() error) error {
	for {
		err := fn()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (nwk *Network) recvLoop(ctx context.Context) error {
	for {
		var src inet256.Addr
		var msg Message
		var validMsg bool
		if err := nwk.swarm.Receive(ctx, func(m p2p.Message[inet256.Addr]) {
			if err := json.Unmarshal(m.Payload, &msg); err != nil {
				nwk.log.Warn("error parsing message from: ", m.Src, err)
				return
			}
			src = m.Src
			validMsg = true
		}); err != nil {
			return err
		}
		if !validMsg {
			continue
		}
		go func() {
			ctx, cf := context.WithTimeout(ctx, 10*time.Second)
			defer cf()
			if err := nwk.fromBelow(ctx, src, msg); err != nil {
				nwk.log.Error("error handling message ", err)
			}
		}()
	}
}

func (n *Network) fromBelow(ctx context.Context, from inet256.Addr, msg Message) error {
	pubKey, err := inet256.ParsePublicKey(msg.SrcKey)
	if err != nil {
		return err
	}
	n.addPeer(pubKey)
	src, err := msg.GetSrc()
	if err != nil {
		return err
	}

	switch msg.Mode {
	case modeSolicit:
		msgAdv := newMessage(n.privateKey, src, nil, maxHops, modeAdvertise)
		data, err := json.Marshal(msgAdv)
		if err != nil {
			panic(err)
		}
		if err := n.swarm.Tell(ctx, from, p2p.IOVec{data}); err != nil {
			return err
		}

	case modeAdvertise:
	case modeData:
		if bytes.Equal(n.localAddr[:], msg.Dst) {
			return n.recvHub.Deliver(ctx, p2p.Message[inet256.Addr]{
				Src:     src,
				Dst:     n.localAddr,
				Payload: msg.Payload,
			})
		}
	}
	return n.forward(ctx, from, msg)
}

func (n *Network) addPeer(pubKey inet256.PublicKey) {
	src := inet256.NewAddr(pubKey)
	n.mu.Lock()
	n.peers[src] = pubKey
	n.mu.Unlock()
}

func (n *Network) forward(ctx context.Context, from inet256.Addr, msg Message) error {
	// forward
	if bytes.Equal(n.localAddr[:], msg.Dst) {
		return nil
	}
	msg2 := msg
	msg2.Hops--
	if msg2.Hops < 0 {
		return nil
	}
	dst := msg.GetDst()
	data, err := json.Marshal(msg2)
	if err != nil {
		panic(err)
	}
	if n.onehop.Contains(dst) {
		return n.swarm.Tell(ctx, dst, p2p.IOVec{data})
	}
	return n.broadcast(ctx, data, from)
}

func (n *Network) broadcast(ctx context.Context, data []byte, exclude inet256.Addr) error {
	eg := errgroup.Group{}
	for _, id := range n.onehop.ListPeers() {
		if id == exclude {
			continue
		}
		id := id
		eg.Go(func() error {
			return n.swarm.Tell(ctx, id, p2p.IOVec{data})
		})
	}
	return eg.Wait()
}

func (n *Network) solicit(ctx context.Context) error {
	msg := newMessage(n.privateKey, Addr{}, nil, 10, 1)
	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return n.broadcast(ctx, data, n.localAddr)
}
