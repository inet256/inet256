package inet256srv

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/p2pmux"
	"github.com/brendoncarroll/go-p2p/s/fragswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
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
	Swarms   map[string]p2p.SecureSwarm
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
	transportSwarm := multiswarm.NewSecure(params.Swarms)
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

func (n *node) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	return n.network.LookupPublicKey(ctx, target)
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
