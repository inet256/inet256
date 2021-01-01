package kadsrnet

import (
	"bytes"
	"context"
	"log"
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
	Overhead       = 0 + MaxPathLen*4 + maxSigSize + timestampSize + protobufFields
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
	peerInfos := newPeerInfoStore(privateKey.Public(), swarm, peers, 128)
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
	n.OnRecv(nil)
	n.pinger = newPinger(n.sendPing)
	n.infoSrv = newInfoService(peerInfos, routeTable, n.sendLookupPeer)
	n.crawler = newCrawler(n.routeTable, n.pinger, n.infoSrv, n.sendQueryRoutes)
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

func (n *Network) fromBelow(msg *p2p.Message, next inet256.RecvFunc) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second)
	defer cf()
	kmsg := &Message{}
	if err := proto.Unmarshal(msg.Payload, kmsg); err != nil {
		n.log.Error("garbage: ", err)
		return
	}
	up, err := n.handleMessage(ctx, kmsg)
	if err != nil {
		n.log.Error("while handling message: ", err)
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
	msg := &Message{
		Src:  n.localAddr[:],
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
	route := n.routeTable.RouteTo(dst)
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
			log.Println("invalid: ", msg)
			return err
		}
		// if not addressed to us, forward.
		if !bytes.HasPrefix(msg.Dst[:], n.localAddr[:]) {
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
	log := n.log.WithFields(logrus.Fields{"src": src})
	switch body := body.GetBody().(type) {
	case *Body_Data:
		log.Debug("recv data")
		return body.Data, nil
	case *Body_RouteList:
		log.Debugf("recv RouteList: len=%d", len(body.RouteList.Routes))
		err := n.crawler.onRoutes(ctx, src, body.RouteList.Routes)
		return nil, err
	case *Body_QueryRoutes:
		log.Debugf("recv QueryRoutes: limit=%d", body.QueryRoutes.Limit)
		routeList, err := n.crawler.onQueryRoutes(body.QueryRoutes.Locus, int(body.QueryRoutes.Nbits), int(body.QueryRoutes.Limit))
		if err != nil {
			return nil, err
		}
		err = n.sendRouteList(ctx, src, routeList)
		return nil, err
	case *Body_LookupPeerReq:
		log.Debug("recv LookupPeer")
		res := n.infoSrv.onLookupPeer(body.LookupPeerReq)
		if res == nil {
			return nil, nil
		}
		return nil, n.sendLookupPeerRes(ctx, src, res)
	case *Body_LookupPeerRes:
		log.Debug("recv PeerInfo")
		return nil, n.infoSrv.onLookupPeerRes(body.LookupPeerRes)
	case *Body_Ping:
		log.Debug("recv Ping")
		pong := n.pinger.onPing(src, body.Ping)
		return nil, n.sendPong(ctx, src, pong)
	case *Body_Pong:
		n.pinger.onPong(src, body.Pong)
		return nil, nil
	default:
		return nil, errors.Errorf("empty message")
	}
}

func (n *Network) forward(ctx context.Context, msg *Message) error {
	if len(msg.Path) > MaxPathLen {
		return errors.Errorf("msg path exceeds maximum length")
	}
	ourRoute := n.routeTable.Get(idFromBytes(msg.Dst))
	// try to find an exact path
	// ping is used to test paths, so it must be fully specified, and not changed.
	if !bodyIsPing(msg.Body) {
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
		return errors.Errorf("link index is invalid %d", next)
	}
	msg.Path = msg.Path[:len(msg.Path)-1]
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return n.swarm.Tell(ctx, next, data)
}

func (n *Network) sendLookupPeerRes(ctx context.Context, dst Addr, res *LookupPeerRes) error {
	return n.sendBody(ctx, dst, &Body{
		Body: &Body_LookupPeerRes{
			LookupPeerRes: res,
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
		Src:  n.localAddr[:],
		Dst:  dst[:],
		Path: path[1:],
		Body: bodyBytes,
	}
	nextPeer := n.linkMap.Int2ID(path[0])
	return n.sendExactMessage(ctx, nextPeer, msg)
}

func (n *Network) sendPong(ctx context.Context, dst Addr, pong *Pong) error {
	return n.sendBody(ctx, dst, &Body{
		Body: &Body_Pong{
			Pong: pong,
		},
	})
}

func (n *Network) sendLookupPeer(ctx context.Context, dst Addr, lp *LookupPeerReq) error {
	return n.sendBody(ctx, dst, &Body{
		Body: &Body_LookupPeerReq{
			LookupPeerReq: lp,
		},
	})
}

func (n *Network) totalPeerSet() inet256.PeerSet {
	return n.oneHopPeers
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
