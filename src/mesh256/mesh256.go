// package mesh256 implements an INET256 Service in terms of a distributed routing algorithm
// and a dynamic set of one hop connections to peers.
package mesh256

import (
	"fmt"

	"github.com/cloudflare/circl/sign/ed25519"
	"go.brendoncarroll.net/p2p/f/x509"
	"go.brendoncarroll.net/p2p/s/multiswarm"
	"go.inet256.org/inet256/src/inet256"
	"go.inet256.org/inet256/src/internal/peers"
)

type (
	Addr          = inet256.Addr
	Node          = inet256.Node
	TransportAddr = multiswarm.Addr

	PeerSet = peers.Set
)

const (
	TransportMTU = (1 << 17)
)

func NewPeerStore() peers.Store[TransportAddr] {
	return peers.NewStore[TransportAddr]()
}

func PublicKeyFromX509(x x509.PublicKey) (inet256.PublicKey, error) {
	switch x.Algorithm {
	case x509.Algo_Ed25519:
		return ed25519.PublicKey(x.Data), nil
	default:
		return nil, fmt.Errorf("cannot create INET256 public key from x509 algo=%v", x.Algorithm)
	}
}

func PublicKeyFromINET256(x inet256.PublicKey) x509.PublicKey {
	switch x := x.(type) {
	case ed25519.PublicKey:
		data, err := x.MarshalBinary()
		if err != nil {
			panic(err)
		}
		return x509.PublicKey{
			Algorithm: x509.Algo_Ed25519,
			Data:      data,
		}
	default:
		panic(fmt.Sprintf("cannot convert type %T to x509", x))
	}
}
