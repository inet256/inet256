package mesh256

import (
	"bytes"
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/f/x509"
	"github.com/brendoncarroll/go-p2p/s/p2pkeswarm"
	"github.com/brendoncarroll/stdctx/logctx"

	"github.com/inet256/inet256/pkg/inet256"
)

// identitySwarm turns p2pkeswarm.Addr[inet256.Addr] into inet256.Addr
type identitySwarm struct {
	*p2pkeswarm.Swarm[inet256.Addr]
}

func newIdentitySwarm(privateKey inet256.PrivateKey, insecure p2p.Swarm[inet256.Addr]) identitySwarm {
	fingerprinter := func(pubKey *x509.PublicKey) p2p.PeerID {
		pubKey2, err := PublicKeyFromX509(*pubKey)
		if err != nil {
			// TODO
			panic(err)
		}
		return p2p.PeerID(inet256.NewAddr(pubKey2))
	}
	privX509 := convertINET256PrivateKey(privateKey)
	quicSw := p2pkeswarm.New(insecure, privX509, p2pkeswarm.WithFingerprinter[inet256.Addr](fingerprinter))
	return identitySwarm{Swarm: quicSw}
}

func (s identitySwarm) Receive(ctx context.Context, th func(p2p.Message[inet256.Addr])) error {
	for done := false; !done; {
		if err := s.Swarm.Receive(ctx, func(msg p2p.Message[p2pkeswarm.Addr[inet256.Addr]]) {
			srcID := msg.Src.ID
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

func (s identitySwarm) LookupPublicKey(ctx context.Context, target inet256.Addr) (x509.PublicKey, error) {
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

var x509Registry = x509.DefaultRegistry()

func convertINET256PrivateKey(x inet256.PrivateKey) x509.PrivateKey {
	switch x := x.(type) {
	case *inet256.Ed25519PrivateKey:
		return x509.PrivateKey{
			Algorithm: x509.Algo_Ed25519,
			Data:      x.Seed(),
		}
	default:
		return x509.PrivateKey{}
	}
}
