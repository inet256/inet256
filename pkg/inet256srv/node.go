package inet256srv

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/p2pmux"
	"github.com/brendoncarroll/go-p2p/s/fragswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
)

type PeerStore = inet256.PeerStore
type PeerSet = inet256.PeerSet
type NetworkSpec = inet256.NetworkSpec
type Node = inet256.Node
type Network = inet256.Network
type Addr = inet256.Addr
type NetworkParams = inet256.NetworkParams

type Params struct {
	p2p.PrivateKey
	Swarms   map[string]p2p.SecureSwarm
	Peers    PeerStore
	Networks []NetworkSpec
}

type node struct {
	params Params

	transportSwarm p2p.SecureSwarm
	basePeerSwarm  *peerSwarm
	network        Network
}

func NewNode(params Params) Node {
	transportSwarm := multiswarm.NewSecure(params.Swarms)
	basePeerSwarm := newPeerSwarm(transportSwarm, params.Peers)
	mux := p2pmux.NewVarintSecureMux(basePeerSwarm)

	// create multi network
	networks := make([]Network, len(params.Networks))
	for i, nspec := range params.Networks {
		s := mux.Open(nspec.Index)
		s = fragswarm.NewSecure(s, TransportMTU)
		ps := castPeerSwarm(s)
		networks[i] = nspec.Factory(NetworkParams{
			PrivateKey: params.PrivateKey,
			Swarm:      ps,
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

func (n *node) Recv(ctx context.Context, src, dst *Addr, buf []byte) (int, error) {
	return n.network.Recv(ctx, src, dst, buf)
}

func (n *node) WaitRecv(ctx context.Context) error {
	return n.network.WaitRecv(ctx)
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

func (n *node) TransportAddrs() (ret []string) {
	for _, addr := range n.transportSwarm.LocalAddrs() {
		data, _ := addr.MarshalText()
		ret = append(ret, string(data))
	}
	return ret
}

func (n *node) LastSeen(id p2p.PeerID) map[string]time.Time {
	return n.basePeerSwarm.LastSeen(id)
}

func (n *node) ListOneHop() []Addr {
	return n.params.Peers.ListPeers()
}

func (n *node) Bootstrap(ctx context.Context) error {
	return n.network.Bootstrap(ctx)
}

func (n *node) Close() (retErr error) {
	if err := n.network.Close(); retErr == nil {
		retErr = err
	}
	if err := n.basePeerSwarm.Close(); err != nil {
		retErr = err
	}
	if err := n.transportSwarm.Close(); err != nil {
		retErr = err
	}
	return retErr
}
