package mesh256

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/f/x509"
	"github.com/brendoncarroll/go-p2p/s/fragswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/brendoncarroll/stdctx"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/inet256/inet256/pkg/peers"
)

// NodeParams configure a Node
type NodeParams struct {
	Background context.Context
	PrivateKey inet256.PrivateKey
	Swarms     map[string]multiswarm.DynSwarm
	Peers      peers.Store[TransportAddr]
	NewNetwork NetworkFactory
}

type node struct {
	params NodeParams

	secureSwarms   map[string]multiswarm.DynSecureSwarm[x509.PublicKey]
	transportSwarm p2p.SecureSwarm[TransportAddr, x509.PublicKey]
	basePeerSwarm  *swarm[TransportAddr]
	network        Network
	topSwarm       p2p.SecureSwarm[inet256.Addr, x509.PublicKey]
}

func NewNode(params NodeParams) Node {
	secureSwarms, err := makeSecureSwarms(params.Swarms, params.PrivateKey)
	if err != nil {
		panic(err)
	}
	transportSwarm := multiswarm.NewSecure(secureSwarms)
	basePeerSwarm := newSwarm(params.Background, transportSwarm, params.Peers)
	fragSw := fragswarm.NewSecure[Addr, inet256.PublicKey](basePeerSwarm, TransportMTU)
	network := params.NewNetwork(NetworkParams{
		PrivateKey: params.PrivateKey,
		Swarm:      swarmFromP2P(fragSw),
		Peers:      params.Peers,
		Background: stdctx.Child(params.Background, "network"),
	})
	topSwarm := wrapNetwork(params.PrivateKey, network)
	return &node{
		params:         params,
		secureSwarms:   secureSwarms,
		transportSwarm: transportSwarm,
		basePeerSwarm:  basePeerSwarm,
		network:        network,
		topSwarm:       topSwarm,
	}
}

func (n *node) Send(ctx context.Context, dst Addr, data []byte) error {
	return n.topSwarm.Tell(ctx, dst, p2p.IOVec{data})
}

func (n *node) Receive(ctx context.Context, fn func(inet256.Message)) error {
	return n.topSwarm.Receive(ctx, func(x p2p.Message[Addr]) {
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
	return n.topSwarm.MTU(ctx, target)
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
	el.Add(n.topSwarm.Close())
	el.Add(n.network.Close())
	el.Add(n.basePeerSwarm.Close())
	el.Add(n.transportSwarm.Close())
	return el.Err()
}

// MakeSecureSwarms ensures that all the swarms in x are secure, or wraps them, to make them secure
// then copies them to y.
func makeSecureSwarms(x map[string]multiswarm.DynSwarm, privateKey inet256.PrivateKey) (map[string]multiswarm.DynSecureSwarm[x509.PublicKey], error) {
	y := make(map[string]multiswarm.DynSecureSwarm[x509.PublicKey], len(x))
	for k, s := range x {
		k, sec := SecureDynSwarm(k, s, convertINET256PrivateKey(privateKey))
		y[k] = sec
	}
	return y, nil
}

// SecureDynSwarm
func SecureDynSwarm(protoName string, s multiswarm.DynSwarm, privateKey x509.PrivateKey) (string, multiswarm.DynSecureSwarm[x509.PublicKey]) {
	if sec, ok := s.(multiswarm.DynSecureSwarm[x509.PublicKey]); ok {
		return protoName, sec
	}
	fingerprinter := func(pubKey x509.PublicKey) p2p.PeerID {
		pubKey2, err := inet256.PublicKeyFromBuiltIn(pubKey)
		if err != nil {
			panic(err)
		}
		return p2p.PeerID(inet256.NewAddr(pubKey2))
	}
	secSw, err := quicswarm.New[p2p.Addr](s, privateKey, quicswarm.WithFingerprinter[p2p.Addr](fingerprinter))
	if err != nil {
		panic(err)
	}
	dynSecSw := multiswarm.WrapSecureSwarm[quicswarm.Addr[p2p.Addr], x509.PublicKey](secSw)
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
		if _, ok := v.(multiswarm.DynSecureSwarm[x509.PublicKey]); !ok {
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

// swarmFromP2P converts a p2p.SecureSwarm[inet256.Addr, inet256.PublicKey] to a Swarm
// Swarm is the type passed to a Network
func swarmFromP2P(x p2p.SecureSwarm[inet256.Addr, inet256.PublicKey]) Swarm {
	return swarmWrapper{x}
}

// swarmWrapper creates a Swarm from a p2p.SecureSwarm
type swarmWrapper struct {
	p2p.SecureSwarm[inet256.Addr, inet256.PublicKey]
}

func (s swarmWrapper) LocalAddr() inet256.Addr {
	return s.SecureSwarm.LocalAddrs()[0]
}

// func (s swarmWrapper) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
// 	pubKey, err := s.SecureSwarm.LookupPublicKey(ctx, target)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return inet256.PublicKeyFromBuiltIn(pubKey)
// }

// func (s swarmWrapper) PublicKey() inet256.PublicKey {
// 	pubKey, err := inet256.PublicKeyFromBuiltIn(s.SecureSwarm.PublicKey())
// 	if err != nil {
// 		panic(err)
// 	}
// 	return pubKey
// }
