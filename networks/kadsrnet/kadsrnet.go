package kadsrnet

import (
	"bytes"
	"context"
	"log"
	"sync"
	"sync/atomic"
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

	pinger   *pinger
	infoSrv  *infoService
	routeSrv *routeService
	crawler  *crawler

	cf context.CancelFunc

	mu     sync.RWMutex
	onRecv inet256.RecvFunc
}

func New(privateKey p2p.PrivateKey, swarm peerswarm.Swarm, peers inet256.PeerSet, log *logrus.Logger) *Network {
	if log == nil {
		log = logrus.New()
	}
	localAddr := swarm.LocalAddrs()[0].(p2p.PeerID)
	ctx, cf := context.WithCancel(context.Background())
	// state
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
	// request/response services
	n.pinger = newPinger(n.sendBodyAlong)
	n.infoSrv = newInfoService(peerInfos, routeTable, n.sendBodyAlong)
	n.routeSrv = newRouteService(n.routeTable, n.sendBodyAlong)

	n.crawler = newCrawler(n.routeTable, n.routeSrv, n.pinger, n.infoSrv, n.log)

	n.swarm.OnTell(n.fromBelow)
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
	n.mu.Lock()
	n.onRecv = fn
	n.mu.Unlock()
}

func (n *Network) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	return Addr{}, nil
}

func (n *Network) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	lookupFuncs := []func(ctx context.Context, target Addr) (p2p.PublicKey, error){
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
		func(ctx context.Context, target Addr) (p2p.PublicKey, error) {
			r := n.routeTable.RouteTo(target)
			if r == nil {
				return nil, inet256.ErrPublicKeyNotFound
			}
			peerInfo, err := n.infoSrv.get(ctx, target, r)
			if err != nil {
				return nil, err
			}
			pubKey, err := p2p.ParsePublicKey(peerInfo.PublicKey)
			if err != nil {
				return nil, err
			}
			return pubKey, nil
		},
	}
	for _, lookupFunc := range lookupFuncs {
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

func (n *Network) WaitReady(ctx context.Context) error {
	const peerCount = 10
	if err := n.crawler.crawlAll(ctx); err != nil {
		return err
	}
	routes, err := readRoutes(n.routeTable, n.localAddr[:], 0, peerCount)
	if err != nil {
		return err
	}
	return n.waitForPeers(ctx, routes, peerCount*7/10)
}

func (n *Network) Routes() (ret []*Route) {
	n.routeTable.ForEach(func(r *Route) error {
		ret = append(ret, r)
		return nil
	})
	return ret
}

func (n *Network) DumpRoutes() {
	log.Println("ROUTES", n.localAddr)
	for i, r := range n.Routes() {
		log.Println(i, idFromBytes(r.Dst), r.Path)
	}
}

func (n *Network) toAbove(src, dst Addr, data []byte) {
	n.mu.RLock()
	onRecv := n.onRecv
	n.mu.RUnlock()
	onRecv(src, dst, data)
}

func (n *Network) fromBelow(msg *p2p.Message) {
	ctx, cf := context.WithTimeout(context.Background(), 10*time.Second)
	defer cf()
	kmsg := &Message{}
	if err := proto.Unmarshal(msg.Payload, kmsg); err != nil {
		n.log.Error("garbage: ", err)
		return
	}
	prev := msg.Src.(Addr)
	if err := n.handleMessage(ctx, prev, kmsg); err != nil {
		n.log.Warn("while handling message: ", err)
		return
	}
}

func (n *Network) sendResponseBody(ctx context.Context, dst, next Addr, retPath Path, body *Body) error {
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		panic(err)
	}
	// reverse path
	p := reversedPath(retPath)
	msg := &Message{
		Src:       n.localAddr[:],
		Dst:       dst[:],
		Body:      bodyBytes,
		Path:      p,
		PathMatch: uint32(len(dst) * 8), // will always be exact match
	}
	return n.sendMessage(ctx, next, msg)
}

// sendBodyTo will
// - lookup a path and next hop peer
// - add the path to the message
// - add a signature for the body, if it originates locally.
// - marshal it
// - send it to the next hop peer
func (n *Network) sendBodyTo(ctx context.Context, dst Addr, body *Body) error {
	route := n.routeTable.RouteTo(dst)
	if route == nil {
		return inet256.ErrAddrUnreachable{Addr: dst}
	}
	return n.sendBodyAlong(ctx, dst, route, body)
}

func (n *Network) sendBodyAlong(ctx context.Context, dst Addr, r *Route, body *Body) error {
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		panic(err)
	}
	path := r.Path
	mb := matchingBits(dst[:], r.Dst)
	nextPeer := n.linkMap.Int2ID(path[0])
	if nextPeer.Equals(p2p.ZeroPeerID()) {
		// maybe panic here, it means we put a bad link into the route table
		return errors.Errorf("route contains invalid link %d", path[0])
	}
	msg := &Message{
		Src: n.localAddr[:],
		Dst: dst[:],

		Path:      path[1:],
		PathMatch: uint32(mb),

		Body: bodyBytes,
	}
	return n.sendMessage(ctx, nextPeer, msg)
}

// sendExactMessage signs msg's body if it originates locally, marshals it, and sends it to next.
func (n *Network) sendMessage(ctx context.Context, next Addr, msg *Message) error {
	// if it is from us, sign it.
	if bytes.Equal(msg.Src, n.localAddr[:]) {
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

func (n *Network) handleMessage(ctx context.Context, prev Addr, msg *Message) error {
	now := time.Now()
	body := &Body{}
	if err := proto.Unmarshal(msg.Body, body); err != nil {
		return err
	}
	publicKey := n.getPublicKey(ctx, msg, body)
	var verified bool
	if publicKey != nil {
		if err := VerifyMessage(publicKey, now, msg); err != nil {
			return err
		}
		verified = true
	}
	// if not addressed to us, forward.
	if !bytes.Equal(n.localAddr[:], msg.Dst) {
		return n.forward(ctx, prev, msg)
	}
	// if addresssed to us, interpret
	if !verified {
		return errors.Errorf("not interpretting invalid message: %v", body)
	}
	return n.handleBody(ctx, prev, msg, body)
}

// getPublicKey tries to extract a public key from the message or retrive one locally
func (n *Network) getPublicKey(ctx context.Context, msg *Message, body *Body) p2p.PublicKey {
	// if the message is from the peer in the peerInfo, use the public key they provided
	var publicKeyBytes []byte
	if peerInfo := body.GetPeerInfo(); peerInfo != nil {
		publicKeyBytes = peerInfo.PublicKey
	} else if peerInfoReq := body.GetPeerInfoReq(); peerInfoReq != nil {
		publicKeyBytes = peerInfoReq.AskerPublicKey
	}
	var publicKey p2p.PublicKey
	if len(publicKeyBytes) > 0 {
		pk, err := p2p.ParsePublicKey(publicKeyBytes)
		if err != nil {
			return err
		}
		peerID := p2p.NewPeerID(pk)
		if !bytes.Equal(peerID[:], msg.Src) {
			return nil
		}
		publicKey = pk
	}
	// if the publicKey is not already set (from the message)
	// then try to look it up.
	if publicKey == nil {
		pk, err := n.LookupPublicKey(ctx, idFromBytes(msg.Src))
		if err != nil && err != p2p.ErrPublicKeyNotFound {
			return err
		}
		publicKey = pk
	}
	return publicKey
}

func (n *Network) handleBody(ctx context.Context, prev Addr, msg *Message, body *Body) (err error) {
	src := idFromBytes(msg.Src)
	//log := n.log.WithFields(logrus.Fields{"src": src, "dst": n.localAddr})
	switch body := body.GetBody().(type) {
	case *Body_Data:
		//log.Debugf("recv data len=%d", len(body.Data))
		n.toAbove(src, n.localAddr, body.Data)
		return nil

	// Routes
	case *Body_QueryRoutes:
		// log.Debugf("recv QueryRoutes: limit=%d", body.QueryRoutes.Limit)
		routeList, err := n.routeSrv.handleQueryRoutes(body.QueryRoutes.Locus, int(body.QueryRoutes.Nbits), int(body.QueryRoutes.Limit))
		if err != nil {
			return err
		}
		res := &Body{
			Body: &Body_RouteList{RouteList: routeList},
		}
		if err := n.sendResponseBody(ctx, src, prev, msg.ReturnPath, res); err != nil {
			return err
		}
		n.enqueue(prev, src, msg.ReturnPath)
		return nil

	case *Body_RouteList:
		// log.Debugf("recv RouteList: len=%d", len(body.RouteList.Routes))
		return n.routeSrv.handleRouteList(src, body.RouteList)

	// PeerInfo
	case *Body_PeerInfoReq:
		// log.Debugf("recv PeerInfoReq")
		peerInfo, err := n.infoSrv.onPeerInfoReq(body.PeerInfoReq)
		if err != nil {
			return err
		}
		if peerInfo == nil {
			return nil
		}
		res := &Body{
			Body: &Body_PeerInfo{PeerInfo: peerInfo},
		}
		if err := n.sendResponseBody(ctx, src, prev, msg.ReturnPath, res); err != nil {
			return err
		}
		n.enqueue(prev, src, msg.ReturnPath)
		return nil

	case *Body_PeerInfo:
		//log.Debug("recv PeerInfo")
		return n.infoSrv.onPeerInfo(body.PeerInfo)

	// Ping
	case *Body_Ping:
		// log.Debug("recv Ping")
		pong := n.pinger.onPing(src, body.Ping)
		res := &Body{
			Body: &Body_Pong{Pong: pong},
		}
		return n.sendResponseBody(ctx, src, prev, msg.ReturnPath, res)

	case *Body_Pong:
		// log.Debug("recv Pong")
		n.pinger.onPong(src, body.Pong)
		return nil

	default:
		return errors.Errorf("empty message body")
	}
}

func (n *Network) forward(ctx context.Context, prev Addr, msg *Message) error {
	if len(msg.Path)+len(msg.ReturnPath) > MaxPathLen {
		return errors.Errorf("msg paths exceed maximum length %d, %d", len(msg.Path), len(msg.ReturnPath))
	}
	// ping is used to test paths, so it must be fully specified, and not changed.
	if !bodyIsPing(msg.Body) {
		if err := n.improveRoute(msg); err != nil {
			return err
		}
	}
	// if we couldn't find a path, drop it.
	if len(msg.Path) == 0 {
		return errors.Errorf("could not forward message, no path %v -> %v", idFromBytes(msg.Src), idFromBytes(msg.Dst))
	}
	nextIndex := msg.Path[0]
	next := n.linkMap.Int2ID(nextIndex)
	if next == p2p.ZeroPeerID() {
		return errors.Errorf("link index is invalid %d", nextIndex)
	}
	msg.Path = msg.Path[1:]

	msg.ReturnPath = append(msg.ReturnPath, n.linkMap.ID2Int(prev))
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}

	return n.swarm.Tell(ctx, next, data)
}

func (n *Network) improveRoute(msg *Message) error {
	ourRoute := n.routeTable.RouteTo(idFromBytes(msg.Dst))
	if ourRoute == nil {
		return nil
	}
	mb := matchingBits(msg.Dst, ourRoute.Dst)
	if mb <= int(msg.PathMatch) {
		return nil
	}

	// if we are the closest then error
	if bytes.Equal(ourRoute.Dst, n.localAddr[:]) {
		return errors.Errorf("dropping message, we are closest")
	}
	msg.Path = append(Path{}, ourRoute.Path...)
	msg.PathMatch = uint32(mb)
	return nil
}

func (n *Network) enqueue(prev, src Addr, retPath Path) {
	if n.routeTable.Get(src) != nil {
		return
	}
	p := reversedPath(append(retPath, n.linkMap.ID2Int(prev)))
	r := &Route{Dst: src[:], Path: p}
	go func() {
		ctx := context.Background()
		if err := n.crawler.investigate(ctx, r); err != nil {
			n.log.Error("investigating: ", err)
		}
	}()
}

func (n *Network) peerHasUs(ctx context.Context, r *Route) (bool, error) {
	routeList, err := n.routeSrv.queryRoutes(ctx, r, &QueryRoutes{
		Locus: n.localAddr[:],
		Limit: 1,
		Nbits: uint32(len(n.localAddr) * 8),
	})
	if err != nil {
		return false, err
	}
	return containsRouteTo(routeList.Routes, n.localAddr), nil
}

func (n *Network) waitUntilPeerHasUs(ctx context.Context, r *Route) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		yes, err := n.peerHasUs(ctx, r)
		if err != nil {
			return err
		}
		if yes {
			return nil
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (n *Network) waitForPeers(ctx context.Context, routes []*Route, min int) error {
	if len(routes) < min {
		min = len(routes)
	}
	var count uint32
	eg := errgroup.Group{}
	for _, r := range routes {
		r := r
		eg.Go(func() error {
			err := n.waitUntilPeerHasUs(ctx, r)
			if err == nil {
				atomic.AddUint32(&count, 1)
			}
			return err
		})
	}
	err := eg.Wait()
	if int(count) >= min {
		return nil
	}
	return errors.Wrapf(err, "waiting for peers, only got %d/%d", count, min)
}

func (n *Network) runLoop(ctx context.Context) {
	eg := errgroup.Group{}
	eg.Go(func() error {
		n.addLinksLoop(ctx)
		return nil
	})
	eg.Go(func() error {
		return n.crawler.run(ctx)
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

func reversedPath(x Path) Path {
	y := make(Path, len(x))
	for i := range x {
		y[i] = x[len(x)-1-i]
	}
	return y
}
