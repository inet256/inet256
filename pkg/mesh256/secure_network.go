package mesh256

import (
	"bytes"
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/p2pkeswarm"
	"github.com/brendoncarroll/stdctx/logctx"

	"github.com/inet256/inet256/pkg/inet256"
)

func newSecureNetwork(privateKey inet256.PrivateKey, x Network) Network {
	fingerprinter := func(pubKey p2p.PublicKey) p2p.PeerID {
		pubKey2, err := inet256.PublicKeyFromBuiltIn(pubKey)
		if err != nil {
			panic(err)
		}
		return p2p.PeerID(inet256.NewAddr(pubKey2))
	}
	insecure := p2pSwarmFromNetwork(x)
	quicSw := p2pkeswarm.New[inet256.Addr](insecure, privateKey.BuiltIn(), p2pkeswarm.WithFingerprinter[inet256.Addr](fingerprinter))
	secnet := newNetworkFromP2PSwarm(identitySwarm{Swarm: quicSw}, x.FindAddr)
	return secnet
}

// identitySwarm turns p2pke.Addr[inet256.Addr] into inet256.Addr
type identitySwarm struct {
	*p2pkeswarm.Swarm[inet256.Addr]
}

func (s identitySwarm) Receive(ctx context.Context, th func(p2p.Message[inet256.Addr])) error {
	for done := false; !done; {
		if err := s.Swarm.Receive(ctx, func(msg p2p.Message[p2pkeswarm.Addr[inet256.Addr]]) {
			srcID := p2p.ExtractPeerID(msg.Src)
			srcAddr := msg.Src.Addr
			// This is where the actual check for who can send as what address happens
			if !bytes.Equal(srcID[:], srcAddr[:]) {
				logctx.Warnf(ctx, "incorrect id=%v for address=%v", srcID, srcAddr)
				return
			}
			th(p2p.Message[inet256.Addr]{
				Src:     srcAddr,
				Dst:     msg.Dst.Addr,
				Payload: msg.Payload,
			})
			done = true
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s identitySwarm) Tell(ctx context.Context, x inet256.Addr, data p2p.IOVec) error {
	dst := s.makeAddr(x)
	return s.Swarm.Tell(ctx, dst, data)
}

func (s identitySwarm) LocalAddr() inet256.Addr {
	return s.LocalAddrs()[0]
}

func (s identitySwarm) LocalAddrs() (ys []inet256.Addr) {
	for _, x := range s.Swarm.LocalAddrs() {
		ys = append(ys, x.Addr)
	}
	return ys
}

func (s identitySwarm) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	x := s.makeAddr(target)
	return s.Swarm.LookupPublicKey(ctx, x)
}

func (s identitySwarm) MTU(ctx context.Context, target inet256.Addr) int {
	return s.Swarm.MTU(ctx, s.makeAddr(target))
}

func (s identitySwarm) ParseAddr(x []byte) (ret inet256.Addr, _ error) {
	return inet256.ParseAddrBase64(x)
}

func (s identitySwarm) makeAddr(addr inet256.Addr) p2pkeswarm.Addr[inet256.Addr] {
	return p2pkeswarm.Addr[inet256.Addr]{
		ID:   p2p.PeerID(addr),
		Addr: addr,
	}
}

// p2pSwarm converts a Swarm to a p2p.Swarm
type p2pSwarm struct {
	Swarm
	extraSwarmMethods
}

func p2pSwarmFromNetwork(x Network) p2p.SecureSwarm[inet256.Addr] {
	return p2pSwarm{Swarm: x}
}

func (s p2pSwarm) LocalAddrs() []inet256.Addr {
	return []inet256.Addr{s.Swarm.LocalAddr()}
}

func (s p2pSwarm) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	pubKey, err := s.Swarm.LookupPublicKey(ctx, target)
	if err != nil {
		return nil, err
	}
	return pubKey.BuiltIn(), nil
}

func (s p2pSwarm) PublicKey() p2p.PublicKey {
	return s.Swarm.PublicKey().BuiltIn()
}

type extraSwarmMethods struct{}

func (extraSwarmMethods) MaxIncomingSize() int {
	return inet256.MaxMTU
}

func (extraSwarmMethods) ParseAddr(data []byte) (inet256.Addr, error) {
	var addr inet256.Addr
	if err := addr.UnmarshalText(data); err != nil {
		return inet256.Addr{}, err
	}
	return addr, nil
}

type FindAddrFunc = func(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error)

var _ Network = &networkFromSwarm{}

// networkFromSwarm implements a Network from a Swarm
type networkFromSwarm struct {
	Swarm
	findAddr FindAddrFunc
}

func newNetworkFromP2PSwarm(x p2p.SecureSwarm[inet256.Addr], findAddr FindAddrFunc) Network {
	return &networkFromSwarm{
		Swarm:    swarmWrapper{x},
		findAddr: findAddr,
	}
}

func (n *networkFromSwarm) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	return n.findAddr(ctx, prefix, nbits)
}

func (n *networkFromSwarm) MTU(ctx context.Context, target inet256.Addr) int {
	return n.Swarm.MTU(ctx, target)
}

// NetworkFromSwarm creates a Network from a Swarm
func NetworkFromSwarm(x Swarm, findAddr FindAddrFunc) Network {
	return &networkFromSwarm{
		Swarm:    x,
		findAddr: findAddr,
	}
}
