// package mesh256 implements an INET256 Service in terms of a distributed routing algorithm
// and a dynamic set of one hop connections to peers.
package mesh256

import (
	"go.brendoncarroll.net/p2p/f/x509"
	"go.brendoncarroll.net/p2p/s/multiswarm"
	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/peers"
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
	default:
		pkData := x509.MarshalPublicKey(nil, &x)
		pubKey2, err := inet256.ParsePublicKey(pkData)
		if err != nil {
			return nil, err
		}
		return pubKey2, nil
	}
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
