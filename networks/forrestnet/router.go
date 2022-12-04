package forrestnet

import (
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/brendoncarroll/go-tai64"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/inet256/inet256/networks/neteng"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/linkmap"
	"github.com/inet256/inet256/pkg/mesh256"
)

var _ neteng.Router = &Router{}

type Router struct {
	log          mesh256.Logger
	privateKey   inet256.PrivateKey
	publicKey    inet256.PublicKey
	localID      inet256.ID
	oneHop       mesh256.PeerSet
	getPublicKey func(inet256.ID) inet256.PublicKey

	linkMap   *linkmap.LinkMap
	state     *ForrestState
	infoStore *InfoStore
	dhtNode   *kademlia.DHTNode
}

func NewRouter(log mesh256.Logger) *Router {
	return &Router{log: log}
}

func (r *Router) Reset(privateKey inet256.PrivateKey, peers mesh256.PeerSet, getPublicKey neteng.PublicKeyFunc, now time.Time) {
	r.oneHop = peers
	r.privateKey = privateKey
	r.publicKey = r.privateKey.Public()
	r.localID = inet256.NewAddr(r.publicKey)
	r.linkMap = linkmap.New(r.localID)
	r.getPublicKey = getPublicKey

	r.state = NewForrestState(privateKey)
	r.linkMap = linkmap.New(r.localID)
	r.infoStore = NewInfoStore(r.localID, 128)
	r.dhtNode = kademlia.NewDHTNode(kademlia.DHTNodeParams{
		LocalID:       p2p.PeerID(r.localID),
		PeerCacheSize: 128,
		DataCacheSize: 128,
	})
}

func (r *Router) HandleAbove(dst inet256.Addr, data p2p.IOVec, send neteng.SendFunc) bool {
	return false
}

func (r *Router) HandleBelow(from inet256.Addr, data []byte, send neteng.SendFunc, deliver neteng.DeliverFunc, info neteng.InfoFunc) {
	msg, err := ParseMessage(data)
	if err != nil {
		r.log.Warn("error parsing message from ", from, err)
		return
	}
	if err := func() error {
		mtype := msg.GetType()
		switch mtype {
		case MTypeBeacon:
			return r.handleBeacon(send, from, msg)
		case MTypeData:
			return r.handleData(send, deliver, from, msg)
		case MTypeFindNodeReq:
			return r.handleFindNodeReq(send, from, msg)
		case MTypeFindNodeRes:
			return r.handleFindNodeRes(send, from, msg)
		default:
			return errors.Errorf("unrecognized message type %v", mtype)
		}
	}(); err != nil {
		r.log.Warnf("error handling message %v", err)
	}
}

func (r *Router) Heartbeat(now time.Time, send neteng.SendFunc) {
	bcasts := r.state.Heartbeat(tai64.FromGoTime(now), r.listOutgoing(inet256.ID{}))
	for _, b := range bcasts {
		r.sendBeacon(send, getLeafAddr(b), b)
	}
}

func (r *Router) FindAddr(send neteng.SendFunc, info neteng.InfoFunc, prefix []byte, nbits int) {

}

func (r *Router) LookupPublicKey(send neteng.SendFunc, info neteng.InfoFunc, target inet256.ID) {

}

func (r *Router) handleData(send neteng.SendFunc, deliver neteng.DeliverFunc, from inet256.Addr, msg Message) error {
	var rt RoutingTag
	if err := proto.Unmarshal(msg.GetMetadata(), &rt); err != nil {
		return err
	}
	dst := inet256.AddrFromBytes(rt.Dst)
	src := inet256.AddrFromBytes(rt.Src)
	switch {
	case dst == r.localID:
		deliver(src, msg.GetData())
	case r.oneHop.Contains(dst):
		send(dst, p2p.IOVec{msg})
	default:
		panic("need to forward")
	}
	return nil
}

func (r *Router) handleBeacon(send neteng.SendFunc, from inet256.Addr, msg Message) error {
	b, err := parseBeacon(msg.GetBody())
	if err != nil {
		return err
	}
	pubKey := r.getPublicKey(from)
	peer := Peer{
		PublicKey: pubKey,
		Index:     r.linkMap.ID2Int(from),
		Addr:      from,
	}
	downstream := r.listOutgoing(from)
	beacons, err := r.state.Deliver(peer, b, downstream)
	if err != nil {
		return err
	}
	for _, b2 := range beacons {
		b2 := b2
		dst := getLeafAddr(b2)
		r.sendBeacon(send, dst, b2)
	}
	return nil
}

func (r *Router) handleFindNodeReq(send neteng.SendFunc, dst inet256.Addr, msg Message) error {
	return nil
}

func (r *Router) handleFindNodeRes(send neteng.SendFunc, dst inet256.Addr, msg Message) error {
	return nil
}

func (r *Router) sendBeacon(send neteng.SendFunc, dst inet256.Addr, b *Beacon) {
	data, err := proto.Marshal(b)
	if err != nil {
		panic(err)
	}
	send(dst, p2p.IOVec{data})
}

// func (r *Router) broadcastBeacon(send neteng.SendFunc, exclude inet256.Addr, b *Beacon) {
// 	data, err := proto.Marshal(b)
// 	if err != nil {
// 		panic(err)
// 	}
// 	for _, peer := range r.oneHop.ListPeers() {
// 		if peer == exclude {
// 			continue
// 		}
// 		send(peer, p2p.IOVec{data})
// 	}
// }

// listOutgoing creates a list of peers further from the root than the local node.
func (r *Router) listOutgoing(upstream inet256.Addr) []Peer {
	var peers []Peer
	for _, addr := range r.oneHop.ListPeers() {
		if addr == upstream {
			continue
		}
		pubKey := r.getPublicKey(addr)
		peers = append(peers, Peer{
			Addr:      addr,
			Index:     r.linkMap.ID2Int(addr),
			PublicKey: pubKey,
		})
	}
	return peers
}
