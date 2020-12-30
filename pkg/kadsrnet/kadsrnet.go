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
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const (
	MaxPathLen = 64
)

func Factory(params inet256.NetworkParams) inet256.Network {
	return New(params.PrivateKey, params.Swarm, params.Peers)
}

type Addr = inet256.Addr

type Network struct {
	privateKey  p2p.PrivateKey
	swarm       peerswarm.Swarm
	oneHopPeers inet256.PeerSet
	localID     Addr

	linkMap    *linkMap
	routeTable *RouteTable
	peerInfos  *PeerInfoStore
	pinger     *pinger
	crawler    *crawler
	cf         context.CancelFunc
}

func New(privateKey p2p.PrivateKey, swarm peerswarm.Swarm, peers inet256.PeerSet) *Network {
	localID := swarm.LocalAddrs()[0].(p2p.PeerID)
	ctx, cf := context.WithCancel(context.Background())
	routeTable := NewRouteTable(localID, 128)
	n := &Network{
		privateKey:  privateKey,
		swarm:       swarm,
		oneHopPeers: peers,
		localID:     localID,

		linkMap:    newLinkMap(),
		routeTable: routeTable,
		peerInfos:  newPeerInfoStore(localID, 128),
		cf:         cf,
	}
	n.pinger = newPinger(n.sendPing)
	n.crawler = newCrawler(n.routeTable, n.oneHopPeers, n.linkMap, n.pinger, n.sendQueryRoutes)
	go n.runLoop(ctx)
	return n
}

func (n *Network) Tell(ctx context.Context, addr Addr, data []byte) error {
	body := &Body{Body: &Body_Data{Data: data}}
	return n.sendBody(ctx, addr, body)
}

func (n *Network) OnRecv(fn inet256.RecvFunc) {
	if fn == nil {
		fn = inet256.NoOpRecvFunc
	}
	n.swarm.OnTell(func(msg *p2p.Message) {
		n.fromBelow(msg, fn)
	})
}

func (n *Network) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	return Addr{}, nil
}

func (n *Network) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	peerInfo := n.peerInfos.Get(target)
	return p2p.ParsePublicKey(peerInfo.PublicKey)
}

func (n *Network) MTU(ctx context.Context, target Addr) int {
	return 0
}

func (n *Network) Close() error {
	n.cf()
	return n.swarm.Close()
}

func (n *Network) fromBelow(msg *p2p.Message, next inet256.RecvFunc) {
	ctx := context.TODO()
	kmsg := &Message{}
	if err := proto.Unmarshal(msg.Payload, kmsg); err != nil {
		logrus.Error("garbage: ", err)
		return
	}
	up, err := n.handleMessage(ctx, kmsg)
	if err != nil {
		logrus.Error("error handling message: ", err)
	}
	if up != nil {
		next(msg.Src.(Addr), msg.Dst.(Addr), up)
	}
}

// sendBody marshals body and calls sendMessage
func (n *Network) sendBody(ctx context.Context, dst Addr, body *Body) error {
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		panic(err)
	}
	localID := n.LocalID()
	msg := &Message{
		Src:  localID[:],
		Dst:  dst[:],
		Body: bodyBytes,
	}
	return n.sendMessage(ctx, dst, msg)
}

// sendMessage will
// - lookup a path and next hop peer
// - add the path to the message
// - add a signature for the body, if it originates locally.
// - marshal it
// - send it to the next hop peer
func (n *Network) sendMessage(ctx context.Context, dst Addr, msg *Message) error {
	route := n.lookupRoute(dst)
	if route == nil {
		return inet256.ErrAddrUnreachable
	}
	nextPeer := n.linkMap.Int2ID(route.Path[0])
	if nextPeer.Equals(p2p.ZeroPeerID()) {
		// maybe panic here, it means we put a bad link into the route table
		return errors.Errorf("route contains invalid link %d", route.Path[0])
	}
	msg.Path = route.Path[1:]
	return n.sendExactMessage(ctx, nextPeer, msg)
}

// sendExactMessage signs msg's body if it originates locally, marshals it, and sends it to dst.
func (n *Network) sendExactMessage(ctx context.Context, dst Addr, msg *Message) error {
	// if it is from us, sign it.
	if bytes.HasPrefix(msg.Src, n.localID[:]) {
		now := time.Now()
		if err := SignMessage(n.privateKey, now, msg); err != nil {
			return err
		}
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return n.swarm.TellPeer(ctx, dst, data)
}

func (n *Network) handleMessage(ctx context.Context, msg *Message) ([]byte, error) {
	now := time.Now()
	var up []byte
	if err := func() error {
		body := &Body{}
		if err := proto.Unmarshal(msg.Body, body); err != nil {
			return err
		}
		// if the message is from the peer in the peerInfo, use the public key they provided
		var publicKey p2p.PublicKey
		if peerInfo := body.GetPeerInfo(); peerInfo != nil {
			if err := validatePeerInfo(peerInfo); err != nil {
				return err
			}
			pk, err := p2p.ParsePublicKey(peerInfo.PublicKey)
			if err != nil {
				return err
			}
			peerID := p2p.NewPeerID(publicKey)
			if bytes.Equal(peerID[:], peerInfo.Id) {
				publicKey = pk
			}
		}
		// if the publicKey is not already set (from the message)
		// then try to look it up.
		if publicKey == nil {
			pk, err := n.LookupPublicKey(ctx, idFromBytes(msg.Src))
			if err != nil {
				return err
			}
			publicKey = pk
		}
		if err := VerifyMessage(publicKey, now, msg); err != nil {
			return err
		}
		// if not addressed to us, forward.
		if !bytes.HasPrefix(msg.Dst[:], n.localID[:]) {
			return n.forward(ctx, msg)
		}
		// if addresssed to us, interpret
		up2, err := n.handleBody(ctx, idFromBytes(msg.Src), body)
		if err != nil {
			return err
		}
		up = up2
		return nil
	}(); err != nil {
		return nil, err
	}
	return up, nil
}

func (n *Network) handleBody(ctx context.Context, src Addr, body *Body) ([]byte, error) {
	switch body := body.GetBody().(type) {
	case *Body_Data:
		return body.Data, nil
	case *Body_RouteList:
		err := n.crawler.handleRoutes(ctx, src, body.RouteList.Routes)
		return nil, err
	case *Body_QueryRoutes:
		// TODO: respond with routes
		return nil, nil
	case *Body_LookupPeer:
		prefix := body.LookupPeer.Prefix
		nbits := int(body.LookupPeer.Bits)
		peerInfo := n.peerInfos.Lookup(prefix, nbits)
		if peerInfo == nil {
			return nil, nil
		}
		return nil, n.sendPeerInfo(ctx, src, peerInfo)
	default:
		return nil, nil
	}
}

func (n *Network) forward(ctx context.Context, msg *Message) error {
	if len(msg.Path) > MaxPathLen {
		return errors.Errorf("msg path exceeds maximum length")
	}
	// if there is no path, then lookup one
	if len(msg.Path) == 0 {
		if bodyIsPing(msg.Body) {
			// ping is used to test paths, so it must be fully specified.
			return nil
		}
		r := n.lookupRoute(idFromBytes(msg.Dst))
		if r == nil {
			// if we are the closest, then drop it
			return nil
		}
		msg.Path = r.Path
	}
	nextIndex := msg.Path[len(msg.Path)-1]
	next := n.linkMap.Int2ID(nextIndex)
	if next == p2p.ZeroPeerID() {
		return errors.Errorf("link index is invalid %d", next)
	}
	msg.Path = msg.Path[:len(msg.Path)-1]
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return n.swarm.Tell(ctx, next, data)
}

// lookupRoute returns the route to finalDst, or nil if there is no better node than us.
func (n *Network) lookupRoute(finalDst Addr) *Route {
	if n.oneHopPeers.Contains(finalDst) {
		linkIndex := n.linkMap.ID2Int(finalDst)
		return &Route{
			Dst:  finalDst[:],
			Path: Path{linkIndex},
		}
	}
	r := n.routeTable.Closest(finalDst[:])
	peerDist := [32]byte{}
	kademlia.XORBytes(peerDist[:], r.Dst, finalDst[:])
	ourDist := [32]byte{}
	kademlia.XORBytes(ourDist[:], n.localID[:], finalDst[:])
	// peer must actually be closer
	if bytes.Compare(peerDist[:], ourDist[:]) < 0 {
		return r
	}
	return nil
}

func (n *Network) sendPeerInfo(ctx context.Context, dst Addr, peerInfo *PeerInfo) error {
	return n.sendBody(ctx, dst, &Body{
		Body: &Body_PeerInfo{
			PeerInfo: peerInfo,
		},
	})
}

func (n *Network) sendRouteList(ctx context.Context, dst Addr, routeList *RouteList) error {
	return n.sendBody(ctx, dst, &Body{
		Body: &Body_RouteList{
			RouteList: routeList,
		},
	})
}

func (n *Network) sendQueryRoutes(ctx context.Context, dst Addr, query *QueryRoutes) error {
	return n.sendBody(ctx, dst, &Body{
		Body: &Body_QueryRoutes{
			QueryRoutes: query,
		},
	})
}

// sendPing sends a ping message along the specified path
func (n *Network) sendPing(ctx context.Context, dst Addr, path Path, ping *Ping) error {
	body := &Body{
		Body: &Body_Ping{
			Ping: ping,
		},
	}
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		panic(err)
	}
	msg := &Message{
		Src:  n.localID[:],
		Dst:  dst[:],
		Path: path[1:],
		Body: bodyBytes,
	}
	nextPeer := n.linkMap.Int2ID(path[0])
	return n.sendExactMessage(ctx, nextPeer, msg)
}

func (n *Network) totalPeerSet() inet256.PeerSet {
	return n.oneHopPeers
}

func (n *Network) LocalID() p2p.PeerID {
	return n.localID
}

func (n *Network) runLoop(ctx context.Context) {
	eg := errgroup.Group{}
	eg.Go(func() error {
		n.addLinksLoop(ctx)
		return nil
	})
	eg.Go(func() error {
		n.crawler.run(ctx)
		return nil
	})
	eg.Wait()
}

func (n *Network) addLinksLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		for _, peerID := range n.oneHopPeers.ListPeers() {
			n.linkMap.ID2Int(peerID)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func bodyIsPing(data []byte) bool {
	body := Body{}
	if err := proto.Unmarshal(data, &body); err != nil {
		return false
	}
	return body.GetPing() != nil
}
