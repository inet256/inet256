package inet256

import (
	"crypto/x509"
	"math/bits"

	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"
)

type (
	Addr      = p2p.PeerID
	PublicKey = p2p.PublicKey
)

func NewAddr(pubKey PublicKey) Addr {
	return p2p.NewPeerID(pubKey)
}

func AddrFromBytes(x []byte) Addr {
	y := Addr{}
	copy(y[:], x)
	return y
}

func ParsePublicKey(data []byte) (PublicKey, error) {
	return p2p.ParsePublicKey(data)
}

func MarshalPublicKey(pubKey PublicKey) []byte {
	return p2p.MarshalPublicKey(pubKey)
}

func ParsePrivateKey(data []byte) (p2p.PrivateKey, error) {
	privKey, err := x509.ParsePKCS8PrivateKey(data)
	if err != nil {
		return nil, err
	}
	privKey2, ok := privKey.(p2p.PrivateKey)
	if !ok {
		return nil, errors.Errorf("unsupported private key type")
	}
	return privKey2, nil
}

func MarshalPrivateKey(privKey p2p.PrivateKey) ([]byte, error) {
	return x509.MarshalPKCS8PrivateKey(privKey)
}

func HasPrefix(addr Addr, prefix []byte, nbits int) bool {
	if len(addr) < len(prefix) {
		return false
	}
	xor := make([]byte, len(addr))
	for i := range prefix {
		xor[i] = addr[i] ^ prefix[i]
	}
	lz := 0
	for i := range xor {
		lzi := bits.LeadingZeros8(xor[i])
		lz += lzi
		if lzi < 8 {
			break
		}
	}
	return lz == nbits
}
