package floodnet

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"golang.org/x/sync/errgroup"
)

func Factory(params inet256.NetworkParams) inet256.Network {
	return New(params.PrivateKey, params.Swarm, params.Peers, params.Logger)
}

const maxHops = 10

type Addr = inet256.Addr

type Network struct {
	localAddr  inet256.Addr
	privateKey p2p.PrivateKey
	onehop     inet256.PeerSet
	swarm      peerswarm.Swarm
	log        *inet256.Logger

	mu    sync.RWMutex
	peers map[p2p.PeerID]p2p.PublicKey

	recvHub *inet256srv.TellHub
}

func New(privateKey p2p.PrivateKey, ps peerswarm.Swarm, onehop inet256.PeerSet, log *inet256.Logger) inet256.Network {
	n := &Network{
		localAddr:  inet256.NewAddr(privateKey.Public()),
		privateKey: privateKey,
		onehop:     onehop,
		swarm:      ps,
		log:        log,

		peers:   make(map[p2p.PeerID]p2p.PublicKey),
		recvHub: inet256srv.NewTellHub(),
	}
	go func() {
		if err := n.recvLoop(context.Background()); err != nil {
			n.log.Error("exiting recvLoop with", err)
		}
	}()
	return n
}

func (n *Network) Tell(ctx context.Context, dst Addr, data []byte) error {
	msg := newMessage(n.privateKey, dst, data, maxHops, modeData)
	msgData, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return n.broadcast(ctx, msgData, n.localAddr)
}

func (n *Network) Recv(ctx context.Context, src, dst *inet256.Addr, buf []byte) (int, error) {
	return n.recvHub.Recv(ctx, src, dst, buf)
}

func (n *Network) WaitRecv(ctx context.Context) error {
	return n.recvHub.Wait(ctx)
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

func (n *Network) Bootstrap(ctx context.Context) error {
	return n.solicit(ctx)
}

func (n *Network) MTU(ctx context.Context, target Addr) int {
	return n.swarm.MTU(ctx, target)
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
	buf := make([]byte, nwk.swarm.MaxIncomingSize())
	for {
		var src, dst p2p.Addr
		n, err := nwk.swarm.Recv(ctx, &src, &dst, buf)
		if err != nil {
			return err
		}
		msg := Message{}
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			nwk.log.Warn("error parsing message from: ", src)
			continue
		}
		go func() {
			ctx, cf := context.WithTimeout(ctx, 10*time.Second)
			defer cf()
			if err := nwk.fromBelow(ctx, src.(inet256.Addr), msg); err != nil {
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
		if err := n.swarm.TellPeer(ctx, from, p2p.IOVec{data}); err != nil {
			return err
		}

	case modeAdvertise:
	case modeData:
		if bytes.Equal(n.localAddr[:], msg.Dst) {
			return n.recvHub.Deliver(ctx, inet256.Message{
				Src:     src,
				Dst:     n.localAddr,
				Payload: msg.Payload,
			})
		}
	}
	return n.forward(ctx, from, msg)
}

func (n *Network) addPeer(pubKey p2p.PublicKey) {
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
		return n.swarm.TellPeer(ctx, dst, p2p.IOVec{data})
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
			return n.swarm.TellPeer(ctx, id, p2p.IOVec{data})
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
