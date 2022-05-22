package mesh256

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/fragswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/inet256/inet256/pkg/peers"
	"github.com/sirupsen/logrus"
)

type Params struct {
	PrivateKey p2p.PrivateKey
	Swarms     map[string]multiswarm.DynSwarm
	Peers      peers.Store[TransportAddr]
	NewNetwork NetworkFactory
}

type node struct {
	params Params

	secureSwarms   map[string]multiswarm.DynSecureSwarm
	transportSwarm p2p.SecureSwarm[TransportAddr]
	basePeerSwarm  *swarm[TransportAddr]
	network        Network
}

func NewNode(params Params) Node {
	secureSwarms, err := makeSecureSwarms(params.Swarms, params.PrivateKey)
	if err != nil {
		panic(err)
	}
	transportSwarm := multiswarm.NewSecure(secureSwarms)
	basePeerSwarm := newSwarm(transportSwarm, params.Peers)
	fragSw := fragswarm.NewSecure[Addr](basePeerSwarm, TransportMTU)
	nw := params.NewNetwork(NetworkParams{
		PrivateKey: params.PrivateKey,
		Swarm:      swarmFromP2P(fragSw),
		Peers:      params.Peers,
		Logger:     logrus.StandardLogger(),
	})
	network := newChainNetwork(
		newLoopbackNetwork(params.PrivateKey.Public()),
		newSecureNetwork(params.PrivateKey, nw),
	)
	return &node{
		params:         params,
		secureSwarms:   secureSwarms,
		transportSwarm: transportSwarm,
		basePeerSwarm:  basePeerSwarm,
		network:        network,
	}
}

func (n *node) Send(ctx context.Context, dst Addr, data []byte) error {
	return n.network.Tell(ctx, dst, p2p.IOVec{data})
}

func (n *node) Receive(ctx context.Context, fn func(inet256.Message)) error {
	return n.network.Receive(ctx, func(x p2p.Message[Addr]) {
		fn(inet256.Message{
			Src:     x.Src,
			Dst:     x.Dst,
			Payload: x.Payload,
		})
	})
}

func (n *node) FindAddr(ctx context.Context, prefix []byte, nbits int) (addr Addr, err error) {
	return n.network.FindAddr(ctx, prefix, nbits)
}

func (n *node) LocalAddr() Addr {
	return inet256.NewAddr(n.params.PrivateKey.Public())
}

func (n *node) LookupPublicKey(ctx context.Context, target Addr) (inet256.PublicKey, error) {
	return n.network.LookupPublicKey(ctx, target)
}

func (n *node) PublicKey() inet256.PublicKey {
	return n.network.PublicKey()
}

func (n *node) MTU(ctx context.Context, target Addr) int {
	return n.network.MTU(ctx, target)
}

func (n *node) TransportAddrs() []TransportAddr {
	return n.transportSwarm.LocalAddrs()
}

func (n *node) LastSeen(id inet256.Addr) map[string]time.Time {
	return n.basePeerSwarm.LastSeen(id)
}

func (n *node) ListOneHop() []Addr {
	return n.params.Peers.ListPeers()
}

func (n *node) Close() (retErr error) {
	var el netutil.ErrList
	el.Add(n.network.Close())
	el.Add(n.basePeerSwarm.Close())
	el.Add(n.transportSwarm.Close())
	return el.Err()
}

// MakeSecureSwarms ensures that all the swarms in x are secure, or wraps them, to make them secure
// then copies them to y.
func makeSecureSwarms(x map[string]multiswarm.DynSwarm, privateKey p2p.PrivateKey) (map[string]multiswarm.DynSecureSwarm, error) {
	y := make(map[string]multiswarm.DynSecureSwarm, len(x))
	for k, s := range x {
		k, sec := SecureDynSwarm(k, s, privateKey)
		y[k] = sec
	}
	return y, nil
}

// SecureDynSwarm
func SecureDynSwarm(protoName string, s multiswarm.DynSwarm, privateKey p2p.PrivateKey) (string, multiswarm.DynSecureSwarm) {
	if sec, ok := s.(multiswarm.DynSecureSwarm); ok {
		return protoName, sec
	}
	fingerprinter := func(pubKey inet256.PublicKey) p2p.PeerID {
		return p2p.PeerID(inet256.NewAddr(pubKey))
	}
	secSw, err := quicswarm.New[p2p.Addr](s, privateKey, quicswarm.WithFingerprinter[p2p.Addr](fingerprinter))
	if err != nil {
		panic(err)
	}
	dynSecSw := multiswarm.WrapSecureSwarm[quicswarm.Addr[p2p.Addr]](secSw)
	return SecureProtocolName(protoName), dynSecSw
}

// SecureProtocolName returns the name of the protocol after securing an insecure swarm
// It will match what is returned by Server.TransportAddrs()
func SecureProtocolName(x string) string {
	return "quic+" + x
}

func NewAddrSchema(swarms map[string]multiswarm.DynSwarm) multiswarm.AddrSchema {
	swarms2 := map[string]multiswarm.DynSwarm{}
	for k, v := range swarms {
		if _, ok := v.(multiswarm.DynSecureSwarm); !ok {
			k = SecureProtocolName(k)
			v = secureAddrParser{v}
		}
		swarms2[k] = v
	}
	return multiswarm.NewSchemaFromSwarms(swarms2)
}

type secureAddrParser struct {
	multiswarm.DynSwarm
}

func (sap secureAddrParser) ParseAddr(x []byte) (p2p.Addr, error) {
	return quicswarm.ParseAddr(sap.DynSwarm.ParseAddr, x)
}
