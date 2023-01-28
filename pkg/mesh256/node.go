package mesh256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/f/x509"
	"github.com/brendoncarroll/go-p2p/s/fragswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/brendoncarroll/go-tai64"
	"github.com/brendoncarroll/stdctx"

	"github.com/inet256/inet256/internal/netutil"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256/multihoming"
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

	mhSwarm   *multihoming.Swarm[TransportAddr, x509.PublicKey, inet256.ID]
	fragSwarm p2p.SecureSwarm[Addr, x509.PublicKey]
	// network is an instance of a routing algorithm
	network Network
	// topSwarm is the swarm on top of the Network, it secures the application traffic
	topSwarm p2p.SecureSwarm[inet256.Addr, x509.PublicKey]
}

// NewNode returns a new INET256 Node
func NewNode(params NodeParams) Node {
	secureSwarms, err := makeSecureSwarms(params.Swarms, params.PrivateKey)
	if err != nil {
		panic(err)
	}
	transportSwarm := multiswarm.NewSecure(secureSwarms)

	mhSwarm := multihoming.New(multihoming.Params[TransportAddr, x509.PublicKey, inet256.Addr]{
		Background: stdctx.Child(params.Background, "multihoming"),
		Inner:      transportSwarm,
		Peers:      params.Peers,
		GroupBy: func(pub x509.PublicKey) (inet256.ID, error) {
			pub2, err := PublicKeyFromX509(pub)
			if err != nil {
				return inet256.ID{}, nil
			}
			return inet256.NewAddr(pub2), nil
		},
	})
	fragSw := fragswarm.NewSecure[Addr, x509.PublicKey](mhSwarm, TransportMTU)
	network := params.NewNetwork(NetworkParams{
		Background: stdctx.Child(params.Background, "network"),
		PrivateKey: params.PrivateKey,
		Swarm:      swarm{fragSw},
		Peers:      params.Peers,
	})
	topSwarm := wrapNetwork(params.PrivateKey, network)
	return &node{
		params:         params,
		secureSwarms:   secureSwarms,
		transportSwarm: transportSwarm,

		mhSwarm:   mhSwarm,
		fragSwarm: fragSw,
		network:   network,
		topSwarm:  topSwarm,
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

func (n *node) LastSeen(id inet256.Addr) map[string]tai64.TAI64N {
	return n.mhSwarm.LastSeen(id)
}

func (n *node) ListOneHop() []Addr {
	return n.params.Peers.ListPeers()
}

func (n *node) Close() (retErr error) {
	var el netutil.ErrList
	el.Add(n.topSwarm.Close())
	el.Add(n.network.Close())
	el.Add(n.fragSwarm.Close())
	el.Add(n.mhSwarm.Close())
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
		pubKey2, err := PublicKeyFromX509(pubKey)
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
