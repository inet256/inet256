package kadsrnet

import (
	"bytes"
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
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
	localID     Addr

	onRecv inet256.RecvFunc

	linkMap    *linkMap
	routeTable *RouteTable
	peerInfos  *PeerInfoStore
	cf         context.CancelFunc
}

func New(swarm peerswarm.Swarm, peers inet256.PeerSet) *Network {
	localID := swarm.LocalAddrs()[0].(p2p.PeerID)
	ctx, cf := context.WithCancel(context.Background())
	n := &Network{
		swarm:       swarm,
		oneHopPeers: peers,
		localID:     localID,

		linkMap:    newLinkMap(),
		routeTable: NewRouteTable(localID, 128),
		peerInfos:  newPeerInfoStore(localID, 128),
		cf:         cf,
	}
	swarm.OnTell(n.fromBelow)
	go n.runLoop(ctx)
	return n
}

func (n *Network) Tell(ctx context.Context, addr Addr, data []byte) error {
	var route *Route
	if n.oneHopPeers.Contains(addr) {
		route = &Route{
			Dst:  addr[:],
			Path: Path{n.linkMap.ID2Int(addr)},
		}
	} else {
		route = n.routeTable.Closest(addr[:])
	}
	if route == nil {
		return inet256.ErrAddrUnreachable
	}
	nextPeer := n.linkMap.Int2ID(route.Path[0])
	if nextPeer.Equals(p2p.ZeroPeerID()) {
		return errors.Errorf("route contains invalid link %d", route.Path[0])
	}
	localID := n.LocalID()
	msg := &Message{
		Src:  localID[:],
		Dst:  addr[:],
		Path: route.Path[1:],
		Body: &Message_Data{
			Data: data,
		},
	}
	msgData, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return n.swarm.TellPeer(ctx, nextPeer, msgData)
}

func (n *Network) OnRecv(fn inet256.RecvFunc) {
	n.onRecv = fn
}

func (n *Network) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	return Addr{}, nil
}

func (n *Network) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	n.peerInfos.Get(target)
	return nil, nil
}

func (n *Network) MTU(ctx context.Context, target Addr) int {
	return 0
}

func (n *Network) Close() error {
	n.cf()
	return nil
}

func (n *Network) fromBelow(msg *p2p.Message) {
	ctx := context.TODO()
	kmsg := &Message{}
	if err := proto.Unmarshal(msg.Payload, kmsg); err != nil {
		logrus.Error(err)
	}
	// if addresssed to us, interpret
	if bytes.Equal(kmsg.Dst[:], n.localID[:]) {
		if err := n.interpret(ctx, kmsg); err != nil {
			logrus.Error(err)
		}
	}
	// otherwise forward
	if err := n.forward(ctx, kmsg); err != nil {
		logrus.Error(err)
	}
}

func (n *Network) interpret(ctx context.Context, msg *Message) error {
	switch body := msg.GetBody().(type) {
	case *Message_Data:
		src := idFromBytes(msg.Src)
		dst := idFromBytes(msg.Dst)
		onRecv := n.onRecv
		onRecv(src, dst, body.Data)
	case *Message_RouteList:
		for _, r := range body.RouteList.Routes {
			n.routeTable.AddRelativeTo(*r, idFromBytes(body.RouteList.Src))
		}
	case *Message_QueryRoutes:
		// TODO: respond with routes
	case *Message_LookupPeer:
		prefix := body.LookupPeer.Prefix
		nbits := int(body.LookupPeer.Bits)
		peerInfo := n.peerInfos.Lookup(prefix, nbits)
		if peerInfo != nil {
			// respond
		}
	}
	return nil
}

func (n *Network) forward(ctx context.Context, msg *Message) error {
	l := len(msg.Path)
	// if there is no path, then lookup one
	if l == 0 {
		r := n.routeTable.Closest(msg.Dst)
		peerDist := make([]byte, len(r.Dst))
		kademlia.XORBytes(peerDist, r.Dst, msg.Dst)
		ourDist := make([]byte, len(r.Dst))
		kademlia.XORBytes(ourDist, n.localID[:], msg.Dst)
		if bytes.Compare(ourDist, peerDist) < 0 {
			// if we are the closest, then drop it
			return nil
		}
		msg.Path = r.Path
	}
	next := n.linkMap.Int2ID(msg.Path[l-1])
	if next == (p2p.PeerID{}) {
		panic("link index is invalid")
	}
	msg.Path = msg.Path[:l-1]
	data, _ := proto.Marshal(msg)
	return n.swarm.TellPeer(ctx, next, data)
}

func (n *Network) totalPeerSet() inet256.PeerSet {
	return n.oneHopPeers
}

func (n *Network) LocalID() p2p.PeerID {
	return n.localID
}

func (n *Network) runLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for _, peerID := range n.oneHopPeers.ListPeers() {
		n.linkMap.ID2Int(peerID)
	}
}
