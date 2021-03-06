package inet256

import (
	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/kademlia"
)

type (
	PublicKey  = p2p.PublicKey
	PrivateKey = p2p.PrivateKey
)

// Addr is an address in an INET256 Network.
// It uniquely identifies a Node.
type Addr [32]byte

// NewAddr creates a new Addr from a PublicKey
func NewAddr(pubKey PublicKey) Addr {
	return Addr(p2p.NewPeerID(pubKey))
}

// AddrFromBytes creates a new address by reading up to 32 bytes from x
// Note that these bytes are not interpretted as a public key, they are interpretted as the raw address.
func AddrFromBytes(x []byte) Addr {
	y := Addr{}
	copy(y[:], x)
	return y
}

// Network implements net.Addr.Network
func (a Addr) Network() string {
	return "INET256"
}

// String implements net.Addr.String
func (a Addr) String() string {
	data, _ := a.MarshalText()
	return string(data)
}

func (a *Addr) UnmarshalText(x []byte) error {
	return (*p2p.PeerID)(a).UnmarshalText(x)
}

func (a Addr) MarshalText() ([]byte, error) {
	return (p2p.PeerID)(a).MarshalText()
}

// GetPeerID implements p2p.HasPeerID
func (a Addr) GetPeerID() p2p.PeerID {
	return p2p.PeerID(a)
}

func ParsePublicKey(data []byte) (PublicKey, error) {
	return p2p.ParsePublicKey(data)
}

func MarshalPublicKey(pubKey PublicKey) []byte {
	return p2p.MarshalPublicKey(pubKey)
}

func HasPrefix(x []byte, prefix []byte, nbits int) bool {
	return kademlia.HasPrefix(x, prefix, nbits)
}
