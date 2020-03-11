package inet256

import (
	"math/bits"

	"github.com/brendoncarroll/go-p2p"
)

type (
	Addr      = p2p.PeerID
	PublicKey = p2p.PublicKey
)

func NewAddr(pubKey PublicKey) Addr {
	return p2p.NewPeerID(pubKey)
}

func ParsePublicKey(data []byte) (PublicKey, error) {
	return p2p.ParsePublicKey(data)
}

func MarshalPublicKey(pubKey PublicKey) []byte {
	return p2p.MarshalPublicKey(pubKey)
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
