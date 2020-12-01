package kadsrnet

import (
	"bytes"
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

func Factory(params inet256.NetworkParams) inet256.Network {
	return New(params.Swarm, params.Peers)
}

type Addr = inet256.Addr

type Network struct {
	swarm       peerswarm.Swarm
	oneHopPeers inet256.PeerSet

	onRecv inet256.RecvFunc

	linkMap    *linkMap
	routeTable *RouteTable
	cf         context.CancelFunc
}

func New(swarm peerswarm.Swarm, peers inet256.PeerSet) *Network {
	ctx, cf := context.WithCancel(context.Background())
	n := &Network{
		swarm:       swarm,
		oneHopPeers: peers,
		linkMap:     newLinkMap,
		routeTable:  NewRouteTable(),
		cf:          cf,
	}
	swarm.OnTell(n.fromBelow)
	go n.runLoop(ctx)
	return n
}

func (n *Network) SendTo(ctx context.Context, addr Addr, data []byte) error {
	if n.oneHopPeers.Contains(addr) {
		return n.swarm.TellPeer(ctx, addr)
	}
}

func (n *Network) OnRecv(fn func(src, dst Addr, data []byte)) {
	n.onRecv = fn
}

func (n *Network) AddrWithPrefix(ctx context.Context, prefix []byte) (Addr, error) {
	return Addr{}, nil
}

func (n *Network) Close() error {
	n.cf()
	return nil
}

func (n *Network) fromBelow(msg *p2p.Message) {
	kmsg := &Message{}
	if err := proto.Unmarshal(msg.Payload, kmsg); err != nil {
		logrus.Error(err)
	}
	// if addresssed to us, interpret
	if bytes.Equal(msg.Dst, n.localID()) {
		n.interpret(msg)
	}
	// otherwise forward
	n.forward(kmsg)
}

func (n *Network) interpret(msg *Message) {
	switch body := msg.GetBody().(type) {
	case *Message_Data:
		src := idFromBytes(msg.Src)
		dst := idFromBytes(msg.Dst)
		onRecv := n.onRecv
		onRecv(src, dst, n.body.Data)
	case *Message_RouteList:
		for _, r := range body.RouteList.Routes {
			n.routeTable.AddRelative(r, idFromBytes(body.RouteList.Src))
		}
	case *Message_QueryRoutes:
		// TODO: respond with routes
	}
}

func (n *Network) forward(msg *Message) {
	l := len(msg.Path)
	// if there is no path, then lookup one
	if l == 0 {
		r := n.routeTable.Closest(msg.Dst)
		peerDist := make([]byte, len(r.Dst))
		kademlia.XORBytes(peerDist, r.Dst, msg.Dst)
		ourDist := make([]byte, len(r.Dst))
		kademlia.XORBytes(ourDist, n.LocalID()[:], msg.Dst)
		if bytes.Compare(ourDist, peerDist) < 0 {
			// if we are the closest, then drop it
			return
		}
		msg.Path = r.Path
	}
	next := n.linkMap.Int2ID(msg.Path[l-1])
	if next == (p2p.PeerID{}) {
		panic("link index is invalid")
	}
	msg.Path = msg.Path[:l-1]
	data, _ := proto.Marshal(msg)
	if err := n.swarm.TellPeer(ctx, next, data); err != nil {
		logrus.Error(err)
	}
}

func (n *Network) totalPeerSet() inet256.PeerSet {
	return n.oneHopPeers
}

func (n *Network) LocalID() p2p.PeerID {
	return n.swarm.LocalAddrs[0].(p2p.PeerID)
}

func (n *Network) runLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)

	for _, peerID := range n.oneHopPeers.ListPeers() {
		n.linkMap.ID2Int(peerID)
	}
}
