package inet256srv

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/p2pmux"
	"github.com/brendoncarroll/go-p2p/s/fragswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type PeerStore = inet256.PeerStore
type PeerSet = inet256.PeerSet
type Node = inet256.Node
type Network = inet256.Network
type Addr = inet256.Addr
type NetworkParams = inet256.NetworkParams

// NetworkSpec is a name associated with a network factory
type NetworkSpec struct {
	Index   uint64
	Name    string
	Factory inet256.NetworkFactory
}

type Params struct {
	p2p.PrivateKey
	Swarms   map[string]p2p.Swarm
	Peers    PeerStore
	Networks []NetworkSpec
}

type node struct {
	params Params

	transportSwarm p2p.SecureSwarm
	basePeerSwarm  *swarm
	network        Network
}

func NewNode(params Params) Node {
	secureSwarms, err := makeSecureSwarms(params.Swarms, params.PrivateKey)
	if err != nil {
		panic(err)
	}
	transportSwarm := multiswarm.NewSecure(secureSwarms)
	basePeerSwarm := newSwarm(transportSwarm, params.Peers)
	mux := p2pmux.NewUint64SecureMux(basePeerSwarm)

	// create multi network
	networks := make([]Network, len(params.Networks))
	for i, nspec := range params.Networks {
		s := mux.Open(nspec.Index)
		s = fragswarm.NewSecure(s, TransportMTU)
		networks[i] = nspec.Factory(NetworkParams{
			PrivateKey: params.PrivateKey,
			Swarm:      swarmWrapper{s},
			Peers:      params.Peers,
			Logger:     logrus.StandardLogger(),
		})
	}
	network := newChainNetwork(
		newLoopbackNetwork(params.PrivateKey.Public()),
		newSecureNetwork(params.PrivateKey,
			newMultiNetwork(networks...),
		),
	)
	return &node{
		params:         params,
		transportSwarm: transportSwarm,
		basePeerSwarm:  basePeerSwarm,
		network:        network,
	}
}

func (n *node) Tell(ctx context.Context, dst Addr, data []byte) error {
	return n.network.Tell(ctx, dst, data)
}

func (n *node) Receive(ctx context.Context, src, dst *Addr, buf []byte) (int, error) {
	return n.network.Receive(ctx, src, dst, buf)
}

func (n *node) WaitReceive(ctx context.Context) error {
	return n.network.WaitReceive(ctx)
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
	return n.params.PrivateKey.Public()
}

func (n *node) MTU(ctx context.Context, target Addr) int {
	return n.network.MTU(ctx, target)
}

func (n *node) TransportAddrs() []p2p.Addr {
	return n.transportSwarm.LocalAddrs()
}

func (n *node) LastSeen(id inet256.Addr) map[string]time.Time {
	return n.basePeerSwarm.LastSeen(id)
}

func (n *node) ListOneHop() []Addr {
	return n.params.Peers.ListPeers()
}

func (n *node) Bootstrap(ctx context.Context) error {
	return n.network.Bootstrap(ctx)
}

func (n *node) Close() (retErr error) {
	var el netutil.ErrList
	el.Do(n.network.Close)
	el.Do(n.basePeerSwarm.Close)
	el.Do(n.transportSwarm.Close)
	return el.Err()
}

// makeSecureSwarms ensures that all the swarms in x are secure, or wraps them, to make them secure
// then copies them to y.
func makeSecureSwarms(x map[string]p2p.Swarm, privateKey p2p.PrivateKey) (map[string]p2p.SecureSwarm, error) {
	fingerprinter := func(pubKey inet256.PublicKey) p2p.PeerID {
		return p2p.PeerID(inet256.NewAddr(pubKey))
	}
	y := make(map[string]p2p.SecureSwarm, len(x))
	for k, s := range x {
		if sec, ok := s.(p2p.SecureSwarm); ok {
			y[k] = sec
		} else {
			var err error
			k = "quic+" + k
			y[k], err = quicswarm.New(s, privateKey, quicswarm.WithFingerprinter(fingerprinter))
			if err != nil {
				return nil, errors.Wrapf(err, "while securing swarm %v", k)
			}
		}

	}
	return y, nil
}
