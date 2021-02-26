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
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func Factory(params inet256.NetworkParams) inet256.Network {
	return New(params.PrivateKey, params.Swarm, params.Peers)
}

const maxHops = 10

type Addr = inet256.Addr

type Network struct {
	localAddr  inet256.Addr
	privateKey p2p.PrivateKey
	onehop     inet256.PeerSet
	swarm      peerswarm.Swarm

	mu     sync.RWMutex
	onRecv inet256.RecvFunc
	peers  map[p2p.PeerID]p2p.PublicKey
}

func New(privateKey p2p.PrivateKey, ps peerswarm.Swarm, onehop inet256.PeerSet) inet256.Network {
	n := &Network{
		localAddr:  inet256.NewAddr(privateKey.Public()),
		privateKey: privateKey,
		onehop:     onehop,
		swarm:      ps,

		onRecv: inet256.NoOpRecvFunc,
		peers:  make(map[p2p.PeerID]p2p.PublicKey),
	}
	ps.OnTell(n.fromBelow)
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

func (n *Network) OnRecv(fn inet256.RecvFunc) {
	if fn == nil {
		fn = inet256.NoOpRecvFunc
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onRecv = fn
}

func (n *Network) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	if err := n.solicit(ctx); err != nil {
		return Addr{}, err
	}
	for i := 0; i < 10; i++ {
		n.mu.RLock()
		for id := range n.peers {
			if inet256.HasPrefix(id[:], prefix, nbits) {
				n.mu.RUnlock()
				return id, nil
			}
		}
		n.mu.RUnlock()
		time.Sleep(500 * time.Millisecond)
	}
	return Addr{}, inet256.ErrNoAddrWithPrefix
}

func (n *Network) LookupPublicKey(ctx context.Context, target Addr) (inet256.PublicKey, error) {
	if err := n.solicit(ctx); err != nil {
		return Addr{}, err
	}
	for i := 0; i < 10; i++ {
		n.mu.RLock()
		for id, pubKey := range n.peers {
			if id == target {
				n.mu.RUnlock()
				return pubKey, nil
			}
		}
		n.mu.RUnlock()
		time.Sleep(500 * time.Millisecond)
	}
	return nil, inet256.ErrPublicKeyNotFound
}

func (n *Network) LocalAddr() Addr {
	return n.localAddr
}

func (n *Network) WaitReady(ctx context.Context) error {
	return n.solicit(ctx)
}

func (n *Network) MTU(ctx context.Context, target Addr) int {
	return n.swarm.MTU(ctx, target)
}

func (n *Network) Close() error {
	return n.swarm.Close()
}

func (n *Network) fromBelow(m *p2p.Message) {
	err := func() error {
		msg := Message{}
		if err := json.Unmarshal(m.Payload, &msg); err != nil {
			return err
		}
		pubKey, err := inet256.ParsePublicKey(msg.SrcKey)
		if err != nil {
			return err
		}
		n.addPeer(pubKey)
		src, err := msg.GetSrc()
		if err != nil {
			return err
		}
		dst := msg.GetDst()

		ctx := context.Background()
		switch msg.Mode {
		case modeSolicit:
			msgAdv := newMessage(n.privateKey, src, nil, maxHops, modeAdvertise)
			data, err := json.Marshal(msgAdv)
			if err != nil {
				panic(err)
			}
			if err := n.broadcast(ctx, data, n.localAddr); err != nil {
				return err
			}
		case modeAdvertise:
		case modeData:
			if bytes.Equal(n.localAddr[:], msg.Dst) {
				n.getOnRecv()(src, n.localAddr, msg.Payload)
				return nil
			}
		}
		// forward
		if bytes.Equal(n.localAddr[:], msg.Dst) {
			return nil
		}
		msg2 := msg
		msg2.Hops--
		if msg2.Hops < 0 {
			return nil
		}
		data, err := json.Marshal(msg2)
		if err != nil {
			panic(err)
		}
		if n.onehop.Contains(dst) {
			return n.swarm.TellPeer(ctx, dst, data)
		}
		return n.broadcast(ctx, data, m.Src.(inet256.Addr))
	}()
	if err != nil {
		logrus.Error(err)
	}
}

func (n *Network) addPeer(pubKey p2p.PublicKey) {
	src := inet256.NewAddr(pubKey)
	n.mu.Lock()
	n.peers[src] = pubKey
	n.mu.Unlock()
}

func (n *Network) getOnRecv() inet256.RecvFunc {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.onRecv != nil {
		return n.onRecv
	}
	return inet256.NoOpRecvFunc
}

func (n *Network) broadcast(ctx context.Context, data []byte, exclude inet256.Addr) error {
	eg := errgroup.Group{}
	for _, id := range n.onehop.ListPeers() {
		if id == exclude {
			continue
		}
		id := id
		eg.Go(func() error {
			return n.swarm.TellPeer(ctx, id, data)
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
