package inet256crypto

import (
	"fmt"

	"golang.org/x/crypto/sha3"
)

// ME256 is a maximum-entropy (meaning all possible values of this type are equally likely)
// value containing 256 bits.
// Each bit is equally likely to be 0 or 1, and different bits are totally uncorrelated.
type ME256 [32]byte

// XOF is an extensible output function
func XOF(salt *ME256, in []byte, out []byte) {
	if salt == nil {
		sha3.ShakeSum256(out, in)
		return
	}
	h := sha3.NewCShake256(nil, salt[:])
	if n, err := h.Read(out); err != nil {
		panic(err)
	} else if n != len(out) {
		panic(fmt.Sprintf("short read from CSHAKE256 n=%d", n))
	}
}

// Sum256 calls XOF and read 256 bits of output.
func Sum256(salt *ME256, in []byte) (ret ME256) {
	XOF(salt, in, ret[:])
	return ret
}
