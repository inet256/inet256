package inet256srv

import (
	"bytes"
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/p2padapter"
	"github.com/sirupsen/logrus"
)

func newSecureNetwork(privateKey inet256.PrivateKey, x Network) Network {
	fingerprinter := func(pubKey inet256.PublicKey) p2p.PeerID {
		return p2p.PeerID(inet256.NewAddr(pubKey))
	}
	insecure := p2padapter.P2PFromINET256(x)
	quicSw, err := quicswarm.New[inet256.Addr](insecure, privateKey, quicswarm.WithFingerprinter[inet256.Addr](fingerprinter))
	if err != nil {
		panic(err)
	}
	secnet := p2padapter.NetworkFromSwarm(quic2Swarm{Swarm: quicSw}, x.FindAddr, x.Bootstrap)
	return secnet
}

// quic2Swarm turns quicswarm.Addr[inet256.Addr] into inet256.Addr
type quic2Swarm struct {
	*quicswarm.Swarm[inet256.Addr]
}

func (s quic2Swarm) Receive(ctx context.Context, th func(p2p.Message[inet256.Addr])) error {
	for done := false; !done; {
		if err := s.Swarm.Receive(ctx, func(msg p2p.Message[quicswarm.Addr[inet256.Addr]]) {
			srcID := p2p.ExtractPeerID(msg.Src)
			srcAddr := (msg.Src).Addr.(inet256.Addr)
			// This is where the actual check for who can send as what address happens
			if !bytes.Equal(srcID[:], srcAddr[:]) {
				logrus.Warnf("incorrect id=%v for address=%v", srcID, srcAddr)
				return
			}
			th(p2p.Message[inet256.Addr]{
				Src:     srcAddr,
				Dst:     msg.Dst.Addr.(inet256.Addr),
				Payload: msg.Payload,
			})
			done = true
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s quic2Swarm) Tell(ctx context.Context, x inet256.Addr, data p2p.IOVec) error {
	dst := s.makeAddr(x)
	return s.Swarm.Tell(ctx, dst, data)
}

func (s quic2Swarm) LocalAddr() inet256.Addr {
	return s.LocalAddrs()[0]
}

func (s quic2Swarm) LocalAddrs() (ys []inet256.Addr) {
	for _, x := range s.Swarm.LocalAddrs() {
		ys = append(ys, x.Addr.(inet256.Addr))
	}
	return ys
}

func (s quic2Swarm) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	x := s.makeAddr(target)
	return s.Swarm.LookupPublicKey(ctx, x)
}

func (s quic2Swarm) MTU(ctx context.Context, target inet256.Addr) int {
	return s.Swarm.MTU(ctx, s.makeAddr(target))
}

func (s quic2Swarm) ParseAddr(x []byte) (ret inet256.Addr, _ error) {
	return inet256.ParseAddrB64(x)
}

func (s quic2Swarm) makeAddr(addr inet256.Addr) quicswarm.Addr[inet256.Addr] {
	return quicswarm.Addr[inet256.Addr]{
		ID:   p2p.PeerID(addr),
		Addr: addr,
	}
}
