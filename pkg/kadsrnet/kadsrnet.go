package kadsrnet

import (
	"bytes"
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const (
	MaxPathLen = 64

	maxSigSize     = 64 // I have no idea
	timestampSize  = 8
	protobufFields = 10
	addrSize       = 32
	Overhead       = 0 + MaxPathLen*4 + maxSigSize + timestampSize + protobufFields + 2*addrSize
)

func Factory(params inet256.NetworkParams) inet256.Network {
	return New(params.PrivateKey, params.Swarm, params.Peers, params.Logger)
}

type Addr = inet256.Addr

type Network struct {
	log         *logrus.Logger
	privateKey  p2p.PrivateKey
	swarm       peerswarm.Swarm
	oneHopPeers inet256.PeerSet
	localAddr   Addr

	linkMap    *linkMap
	routeTable RouteTable
	peerInfos  *PeerInfoStore

	pinger  *pinger
	infoSrv *infoService
	crawler *crawler

	cf context.CancelFunc
}

func New(privateKey p2p.PrivateKey, swarm peerswarm.Swarm, peers inet256.PeerSet, log *logrus.Logger) *Network {
	localAddr := swarm.LocalAddrs()[0].(p2p.PeerID)
	ctx, cf := context.WithCancel(context.Background())
	lm := newLinkMap()
	kadRT := newKadRouteTable(localAddr, 128)
	routeTable := newRouteTable(localAddr, peers, lm, kadRT)
	peerInfos := newPeerInfoStore(privateKey.Public(), swarm, peers, 128*2)
	n := &Network{
		log:         log,
		privateKey:  privateKey,
		swarm:       swarm,
		oneHopPeers: peers,
		localAddr:   localAddr,

		linkMap:    lm,
		routeTable: routeTable,
		peerInfos:  peerInfos,
		cf:         cf,
	}
	n.pinger = newPinger(n.sendBodyAlong)
	n.infoSrv = newInfoService(peerInfos, routeTable, n.sendBodyAlong)
	n.crawler = newCrawler(n.routeTable, n.pinger, n.infoSrv, n.sendQueryRoutes)
	n.OnRecv(nil)
	go n.runLoop(ctx)
	return n
}

func (n *Network) Tell(ctx context.Context, addr Addr, data []byte) error {
	body := &Body{Body: &Body_Data{Data: data}}
	return n.sendBodyTo(ctx, addr, body)
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
	for _, lookupFunc := range []func(ctx context.Context, target Addr) (p2p.PublicKey, error){
		func(ctx context.Context, target Addr) (p2p.PublicKey, error) {
			return n.swarm.LookupPublicKey(ctx, target)
		},
		func(ctx context.Context, target Addr) (p2p.PublicKey, error) {
			peerInfo := n.peerInfos.Get(target)
			if peerInfo == nil {
				return nil, inet256.ErrPublicKeyNotFound
			}
			return p2p.ParsePublicKey(peerInfo.PublicKey)
		},
	} {
		pubKey, err := lookupFunc(ctx, target)
		if err != nil {
			if err == inet256.ErrPublicKeyNotFound {
				continue
			}
			return nil, err
		}
		return pubKey, nil
	}
	return nil, inet256.ErrPublicKeyNotFound
}

func (n *Network) MTU(ctx context.Context, target Addr) int {
	return n.swarm.MTU(ctx, target) - Overhead
}

func (n *Network) LocalAddr() Addr {
	return n.localAddr
}

func (n *Network) Close() error {
	n.cf()
	return n.swarm.Close()
}

func (n *Network) WaitInit(ctx context.Context) error {
	return n.crawler.crawlAll(ctx, 0)
}

func (n *Network) fromBelow(msg *p2p.Message, deliver inet256.RecvFunc) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second)
	defer cf()
	kmsg := &Message{}
	if err := proto.Unmarshal(msg.Payload, kmsg); err != nil {
		n.log.Error("garbage: ", err)
		return
	}
	up, err := n.handleMessage(ctx, msg.Src.(Addr), kmsg)
	if err != nil {
		n.log.Error("while handling message: ", err)
		return
	}
	if up != nil {
		deliver(msg.Src.(Addr), msg.Dst.(Addr), up)
	}
}

func (n *Network) sendResponseBody(ctx context.Context, dst, next Addr, retPath Path, body *Body) error {
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		panic(err)
	}
	// reverse path
	p := make(Path, len(retPath))
	l := len(p)
	for i := 0; i < l/2; i++ {
		p[i], p[l-1-i] = retPath[l-1-i], retPath[i]
	}
	msg := &Message{
		Src:  n.localAddr[:],
		Dst:  dst[:],
		Body: bodyBytes,
		Path: p,
	}
	return n.sendMessage(ctx, next, msg)
}

// sendMessage will
// - lookup a path and next hop peer
// - add the path to the message
// - add a signature for the body, if it originates locally.
// - marshal it
// - send it to the next hop peer
func (n *Network) sendBodyTo(ctx context.Context, dst Addr, body *Body) error {
	route := n.routeTable.RouteTo(dst)
	if route == nil {
		return inet256.ErrAddrUnreachable
	}
	return n.sendBodyAlong(ctx, dst, route.Path, body)
}

func (n *Network) sendBodyAlong(ctx context.Context, dst Addr, path Path, body *Body) error {
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		panic(err)
	}
	nextPeer := n.linkMap.Int2ID(path[0])
	if nextPeer.Equals(p2p.ZeroPeerID()) {
		// maybe panic here, it means we put a bad link into the route table
		return errors.Errorf("route contains invalid link %d", path[0])
	}
	msg := &Message{
		Src: n.localAddr[:],
		Dst: dst[:],

		Path: path[1:],

		Body: bodyBytes,
	}
	return n.sendMessage(ctx, nextPeer, msg)
}

// sendExactMessage signs msg's body if it originates locally, marshals it, and sends it to dst.
func (n *Network) sendMessage(ctx context.Context, next Addr, msg *Message) error {
	// if it is from us, sign it.
	if bytes.HasPrefix(msg.Src, n.localAddr[:]) {
		now := time.Now()
		if err := SignMessage(n.privateKey, now, msg); err != nil {
			return err
		}
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return n.swarm.TellPeer(ctx, next, data)
}

func (n *Network) handleMessage(ctx context.Context, from Addr, msg *Message) ([]byte, error) {
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
			pk, err := p2p.ParsePublicKey(peerInfo.PublicKey)
			if err != nil {
				return err
			}
			peerID := p2p.NewPeerID(pk)
			if bytes.Equal(peerID[:], msg.Src) {
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
		if !bytes.HasPrefix(msg.Dst[:], n.localAddr[:]) {
			return n.forward(ctx, from, msg)
		}
		// if addresssed to us, interpret
		up2, res, err := n.handleBody(ctx, idFromBytes(msg.Src), body)
		if err != nil {
			return err
		}
		if up2 != nil {
			up = up2
		}
		if res != nil {
			return n.sendResponseBody(ctx, idFromBytes(msg.Src), from, msg.ReturnPath, res)
		}
		return nil
	}(); err != nil {
		return nil, err
	}
	return up, nil
}

func (n *Network) handleBody(ctx context.Context, src Addr, body *Body) (up []byte, res *Body, err error) {
	log := n.log.WithFields(logrus.Fields{"src": src, "dst": n.localAddr})
	switch body := body.GetBody().(type) {
	case *Body_Data:
		log.Debugf("recv data len=%d", len(body.Data))
		return body.Data, nil, nil

	// Routes
	case *Body_QueryRoutes:
		log.Debugf("recv QueryRoutes: limit=%d", body.QueryRoutes.Limit)
		routeList, err := n.crawler.onQueryRoutes(body.QueryRoutes.Locus, int(body.QueryRoutes.Nbits), int(body.QueryRoutes.Limit))
		if err != nil {
			return nil, nil, err
		}
		res := &Body{
			Body: &Body_RouteList{RouteList: routeList},
		}
		return nil, res, nil
	case *Body_RouteList:
		log.Debugf("recv RouteList: len=%d", len(body.RouteList.Routes))
		err := n.crawler.onRoutes(ctx, src, body.RouteList.Routes)
		return nil, nil, err

	// PeerInfo
	case *Body_PeerInfoReq:
		log.Debugf("recv PeerInfoReq")
		peerInfo := n.infoSrv.onPeerInfoReq(body.PeerInfoReq)
		if peerInfo == nil {
			return nil, nil, nil
		}
		res := &Body{
			Body: &Body_PeerInfo{PeerInfo: peerInfo},
		}
		return nil, res, nil
	case *Body_PeerInfo:
		log.Debug("recv PeerInfo")
		return nil, nil, n.infoSrv.onPeerInfo(body.PeerInfo)

	// Ping
	case *Body_Ping:
		log.Debug("recv Ping")
		pong := n.pinger.onPing(src, body.Ping)
		res := &Body{
			Body: &Body_Pong{Pong: pong},
		}
		return nil, res, nil
	case *Body_Pong:
		log.Debug("recv Pong")
		n.pinger.onPong(src, body.Pong)
		return nil, nil, nil

	default:
		return nil, nil, errors.Errorf("empty message")
	}
}

func (n *Network) forward(ctx context.Context, prev Addr, msg *Message) error {
	if len(msg.Path)+len(msg.ReturnPath) > MaxPathLen {
		return errors.Errorf("msg paths exceed maximum length")
	}
	// try to find an exact path
	// ping is used to test paths, so it must be fully specified, and not changed.
	if !bodyIsPing(msg.Body) {
		// only replace existing paths with exact matches.
		var ourRoute *Route
		if len(msg.Path) == 0 {
			ourRoute = n.routeTable.RouteTo(idFromBytes(msg.Dst))
		} else {
			ourRoute = n.routeTable.Get(idFromBytes(msg.Dst))
		}
		if ourRoute != nil {
			msg.Path = ourRoute.Path
		}
	}

	// if we couldn't find a path, drop it.
	if len(msg.Path) == 0 {
		return errors.Errorf("could not forward message, no path")
	}
	nextIndex := msg.Path[len(msg.Path)-1]
	next := n.linkMap.Int2ID(nextIndex)
	if next == p2p.ZeroPeerID() {
		return errors.Errorf("link index is invalid %d", nextIndex)
	}
	msg.Path = msg.Path[:len(msg.Path)-1]
	msg.ReturnPath = append(msg.ReturnPath, n.linkMap.ID2Int(prev))
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return n.swarm.Tell(ctx, next, data)
}

func (n *Network) sendQueryRoutes(ctx context.Context, dst Addr, query *QueryRoutes) error {
	if err := n.sendSelfPeerInfo(ctx, dst, nil); err != nil {
		return err
	}
	return n.sendBodyTo(ctx, dst, &Body{
		Body: &Body_QueryRoutes{
			QueryRoutes: query,
		},
	})
}

func (n *Network) sendSelfPeerInfo(ctx context.Context, dst Addr, path Path) error {
	if path == nil {
		r := n.routeTable.RouteTo(dst)
		if r == nil {
			return errors.Errorf("no route to path")
		}
		path = r.Path
	}
	body := &Body{
		Body: &Body_PeerInfo{
			PeerInfo: n.peerInfos.Get(n.localAddr),
		},
	}
	return n.sendBodyAlong(ctx, dst, path, body)
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
