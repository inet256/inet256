package inet256

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"golang.org/x/crypto/sha3"
)

type (
	PublicKey  = crypto.PublicKey
	PrivateKey = crypto.Signer
)

// AddrSize is the size of an address in bytes
const AddrSize = 32

// Addr is an address in an INET256 Network.
// It uniquely identifies a Node.
type Addr [AddrSize]byte

// ID is an alias for Addr
type ID = Addr

// NewAddr creates a new Addr from a PublicKey
func NewAddr(pubKey PublicKey) Addr {
	addr := Addr{}
	sha3.ShakeSum256(addr[:], MarshalPublicKey(pubKey))
	return addr
}

// AddrFromBytes creates a new address by reading up to 32 bytes from x
// Note that these bytes are not interpretted as a public key, they are interpretted as the raw address.
// To derive an address from a PublicKey use NewAddr
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
	n, err := base64.RawURLEncoding.Decode(a[:], x)
	if err != nil {
		return err
	}
	if n != AddrSize {
		return errors.New("too short to be INET256 address")
	}
	return nil
}

func (a Addr) MarshalText() ([]byte, error) {
	return []byte(base64.RawURLEncoding.EncodeToString(a[:])), nil
}

// IsZero returns true if the address is the zero value for the Addr type
func (a Addr) IsZero() bool {
	return a == (Addr{})
}

// ParsePublicKey attempts to parse a PublicKey from data
func ParsePublicKey(data []byte) (PublicKey, error) {
	pubKey, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, err
	}
	switch pubKey.(type) {
	case *rsa.PublicKey, ed25519.PublicKey:
	default:
		return nil, fmt.Errorf("public keys of type %T are not supported by INET256", pubKey)
	}
	return pubKey, nil
}

// MarshalPublicKey marshals pubKey into data.
// MarshalPublicKey panics if it can not marshal the public key.
// All keys returned by ParsePublic key will successfully marshal, so a panic indicates a bug.
func MarshalPublicKey(pubKey PublicKey) []byte {
	data, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		panic(err)
	}
	return data
}

func HasPrefix(x []byte, prefix []byte, nbits int) bool {
	return kademlia.HasPrefix(x, prefix, nbits)
}

// ParseAddrB64 attempts to parse a base64 encoded INET256 address from data
func ParseAddrB64(data []byte) (Addr, error) {
	addr := Addr{}
	err := addr.UnmarshalText(data)
	return addr, err
}
