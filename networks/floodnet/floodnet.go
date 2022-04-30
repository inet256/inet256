package floodnet

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/networks/neteng"
	"github.com/inet256/inet256/pkg/inet256"
)

func Factory(params networks.Params) networks.Network {
	return New(params)
}

const maxHops = 10

type Network struct {
	*neteng.Network
	router *Router
}

func New(params networks.Params) *Network {
	r := NewRouter(params.Logger)
	nwk := neteng.New(params, r, 0)
	return &Network{
		Network: nwk,
		router:  r,
	}
}

type Router struct {
	privateKey inet256.PrivateKey
	publicKey  inet256.PublicKey
	localID    inet256.ID
	peers      networks.PeerSet
	log        networks.Logger

	mu   sync.Mutex
	keys map[inet256.Addr]inet256.PublicKey
}

func NewRouter(log networks.Logger) *Router {
	return &Router{log: log}
}

func (r *Router) Reset(privateKey inet256.PrivateKey, peers networks.PeerSet, getPublicKey neteng.PublicKeyFunc, now time.Time) {
	r.peers = peers
	r.privateKey = privateKey
	r.publicKey = r.privateKey.Public()
	r.localID = inet256.NewAddr(r.publicKey)

	r.keys = make(map[inet256.Addr]inet256.PublicKey)
}

func (r *Router) HandleBelow(from inet256.Addr, data []byte, send neteng.SendFunc, deliver neteng.DeliverFunc, info neteng.InfoFunc) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		r.log.Warn("error parsing message from ", from, err)
		return
	}
	if err := r.handleMessage(send, deliver, info, from, msg); err != nil {
		r.log.Warn("error handling message from ", from, err)
	}
}

func (r *Router) HandleAbove(dst inet256.Addr, data p2p.IOVec, send neteng.SendFunc) bool {
	msg := newMessage(r.privateKey, dst, p2p.VecBytes(nil, data), maxHops, modeData)
	msgData, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	r.broadcast(send, r.localID, p2p.IOVec{msgData})
	return true
}

func (r *Router) FindAddr(send neteng.SendFunc, info neteng.InfoFunc, prefix []byte, nbits int) {
	addr, pubKey := func() (inet256.Addr, inet256.PublicKey) {
		r.mu.Lock()
		defer r.mu.Unlock()
		for id, pubKey := range r.keys {
			if inet256.HasPrefix(id[:], prefix, nbits) {
				return id, pubKey
			}
		}
		return inet256.Addr{}, nil
	}()
	if !addr.IsZero() {
		info(addr, pubKey)
		return
	}
	r.solicit(send)
}

func (r *Router) LookupPublicKey(send neteng.SendFunc, info neteng.InfoFunc, target inet256.Addr) {
	addr, pubKey := func() (inet256.Addr, inet256.PublicKey) {
		r.mu.Lock()
		defer r.mu.Unlock()
		for id, pubKey := range r.keys {
			if id == target {
				return id, pubKey
			}
		}
		return inet256.Addr{}, nil
	}()
	if !addr.IsZero() {
		info(addr, pubKey)
		return
	}
	r.solicit(send)
}

func (r *Router) Heartbeat(now time.Time, send neteng.SendFunc) {}

func (r *Router) handleMessage(send neteng.SendFunc, deliver neteng.DeliverFunc, info neteng.InfoFunc, from inet256.Addr, msg Message) error {
	pubKey, err := inet256.ParsePublicKey(msg.SrcKey)
	if err != nil {
		return err
	}
	r.addPeerInfo(info, pubKey)
	src, err := msg.GetSrc()
	if err != nil {
		return err
	}

	switch msg.Mode {
	case modeSolicit:
		msgAdv := newMessage(r.privateKey, src, nil, maxHops, modeAdvertise)
		data, err := json.Marshal(msgAdv)
		if err != nil {
			panic(err)
		}
		send(from, p2p.IOVec{data})

	case modeAdvertise:
	case modeData:
		if bytes.Equal(r.localID[:], msg.Dst) {
			deliver(src, msg.Payload)
			return nil
		}
	}
	r.forward(send, from, msg)
	return nil
}

func (r *Router) forward(send neteng.SendFunc, from inet256.Addr, msg Message) {
	if bytes.Equal(r.localID[:], msg.Dst) {
		return
	}
	msg2 := msg
	msg2.Hops--
	if msg2.Hops < 0 {
		return
	}
	dst := msg.GetDst()
	data, err := json.Marshal(msg2)
	if err != nil {
		panic(err)
	}
	if r.peers.Contains(dst) {
		send(dst, p2p.IOVec{data})
		return
	}
	r.broadcast(send, from, p2p.IOVec{data})
}

func (r *Router) broadcast(send neteng.SendFunc, exclude inet256.Addr, data p2p.IOVec) {
	for _, id := range r.peers.ListPeers() {
		if id == exclude {
			continue
		}
		send(id, data)
	}
}

func (r *Router) solicit(send neteng.SendFunc) {
	msg := newMessage(r.privateKey, inet256.Addr{}, nil, 10, 1)
	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	r.broadcast(send, r.localID, p2p.IOVec{data})
}

func (r *Router) addPeerInfo(info neteng.InfoFunc, pubKey inet256.PublicKey) {
	src := inet256.NewAddr(pubKey)
	r.mu.Lock()
	r.keys[src] = pubKey
	r.mu.Unlock()
	info(src, pubKey)
}
