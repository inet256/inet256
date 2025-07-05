package inet256

import (
	"encoding/base64"
	"errors"
	"math/bits"

	"golang.org/x/crypto/sha3"
)

const (
	// AddrSize is the size of an address in bytes
	AddrSize = 32
	// Base64Alphabet is used when encoding IDs as base64 strings.
	// It is a URL and filepath safe encoding, which maintains ordering.
	Base64Alphabet = "-0123456789" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "_" + "abcdefghijklmnopqrstuvwxyz"
)

// ID is an address in an INET256 Network.
// It uniquely identifies a Node.
type ID [AddrSize]byte

// Addr is an alias for ID
type Addr = ID

// NewID creates a new Addr from a PublicKey
func NewID(pubKey PublicKey) ID {
	return DefaultPKI.NewID(pubKey)
}

// IDFromBytes creates a new address by reading up to 32 bytes from x
// Note that these bytes are not interpretted as a public key, they are interpretted as the raw address.
// To derive an address from a PublicKey use NewAddr
func IDFromBytes(x []byte) ID {
	y := Addr{}
	copy(y[:], x)
	return y
}

// Network implements net.Addr.Network
func (a ID) Network() string {
	return "INET256"
}

// String implements net.Addr.String
func (a ID) String() string {
	return a.Base64String()
}

// Base64String returns the base64 encoding of the Addr as a string
func (a ID) Base64String() string {
	data, _ := a.MarshalText()
	return string(data)
}

var enc = base64.NewEncoding(Base64Alphabet).WithPadding(base64.NoPadding)

func (a *ID) UnmarshalText(x []byte) error {
	n, err := enc.Decode(a[:], x)
	if err != nil {
		return err
	}
	if n != AddrSize {
		return errors.New("too short to be INET256 address")
	}
	return nil
}

func (a ID) MarshalText() ([]byte, error) {
	return []byte(enc.EncodeToString(a[:])), nil
}

// IsZero returns true if the address is the zero value for the Addr type
func (a Addr) IsZero() bool {
	return a == (Addr{})
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

func Sum256(data []byte) (ret [32]byte) {
	sha3.ShakeSum256(ret[:], data)
	return ret
}
