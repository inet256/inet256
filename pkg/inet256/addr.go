package inet256

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"math/bits"

	"golang.org/x/crypto/sha3"
)

type (
	PublicKey  = crypto.PublicKey
	PrivateKey = crypto.Signer
)

const (
	// AddrSize is the size of an address in bytes
	AddrSize = 32
	// Base64Alphabet is used when encoding IDs as base64 strings.
	// It is a URL and filepath safe encoding, which maintains ordering.
	Base64Alphabet = "-0123456789" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "_" + "abcdefghijklmnopqrstuvwxyz"
)

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
	return a.Base64String()
}

// Base64String returns the base64 encoding of the Addr as a string
func (a Addr) Base64String() string {
	data, _ := a.MarshalText()
	return string(data)
}

var enc = base64.NewEncoding(Base64Alphabet).WithPadding(base64.NoPadding)

func (a *Addr) UnmarshalText(x []byte) error {
	n, err := enc.Decode(a[:], x)
	if err != nil {
		return err
	}
	if n != AddrSize {
		return errors.New("too short to be INET256 address")
	}
	return nil
}

func (a Addr) MarshalText() ([]byte, error) {
	return []byte(enc.EncodeToString(a[:])), nil
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
	var total int
	for i := range prefix {
		if i >= len(x) {
			break
		}
		lz := bits.LeadingZeros8(x[i] ^ prefix[i])
		total += lz
		if total >= nbits {
			return true
		}
		if lz < 8 {
			break
		}
	}
	return false
}

// ParseAddrBase64 attempts to parse a base64 encoded INET256 address from data
func ParseAddrBase64(data []byte) (Addr, error) {
	addr := Addr{}
	err := addr.UnmarshalText(data)
	return addr, err
}
