// package mesh256 implements an INET256 Service in terms of a distributed routing algorithm
// and a dynamic set of one hop connections to peers.
package mesh256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/f/x509"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256/multihoming"
	"github.com/inet256/inet256/pkg/peers"
)

type (
	Addr          = inet256.Addr
	Node          = inet256.Node
	TransportAddr = multiswarm.Addr

	PeerSet = peers.Set
)

const (
	TransportMTU = (1 << 16) - 1

	MinMTU = inet256.MinMTU
	MaxMTU = inet256.MaxMTU
)

func NewPeerStore() peers.Store[TransportAddr] {
	return peers.NewStore[TransportAddr]()
}

func NewSwarm[A p2p.Addr](x p2p.SecureSwarm[A, x509.PublicKey], peers peers.Store[A]) Swarm {
	sw := multihoming.New(multihoming.Params[A, x509.PublicKey, inet256.Addr]{
		Background: context.Background(),
		Inner:      x,
		Peers:      peers,
		GroupBy: func(pub x509.PublicKey) (inet256.Addr, error) {
			pub2, err := PublicKeyFromX509(pub)
			if err != nil {
				return inet256.Addr{}, err
			}
			return inet256.NewAddr(pub2), nil
		},
	})
	return swarm{sw}
}

func PublicKeyFromX509(pubKey x509.PublicKey) (inet256.PublicKey, error) {
	pkData := x509.MarshalPublicKey(nil, &pubKey)
	pubKey2, err := inet256.ParsePublicKey(pkData)
	if err != nil {
		return nil, err
	}
	return pubKey2, nil
}

func PublicKeyFromINET256(x inet256.PublicKey) x509.PublicKey {
	switch x := x.(type) {
	case *inet256.Ed25519PublicKey:
		return x509.PublicKey{
			Algorithm: x509.Algo_Ed25519,
			Data:      []byte(x[:]),
		}
	default:
		pkData := inet256.MarshalPublicKey(nil, x)
		y, err := x509.ParsePublicKey(pkData)
		if err != nil {
			panic(err)
		}
		return y
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
