package inet256crypto

import (
	"crypto"
	"fmt"
	"io"
	"slices"

	"github.com/cloudflare/circl/sign"
)

// ProductSignScheme is a scheme for signing with all of a set of keys
type ProductSignScheme []sign.Scheme

// Name of the scheme.
func (sch ProductSignScheme) Name() string {
	return "product"
}

// GenerateKey creates a new key-pair.
func (sch ProductSignScheme) GenerateKey() (sign.PublicKey, sign.PrivateKey, error) {
	pub := make(ProductVerifier, len(sch))
	priv := make(ProductSigner, len(sch))
	for i := range sch {
		var err error
		pub[i], priv[i], err = sch[i].GenerateKey()
		if err != nil {
			return nil, nil, err
		}
	}
	return pub, priv, nil
}

// Creates a signature using the PrivateKey on the given message and
// returns the signature. opts are additional options which can be nil.
//
// Panics if key is nil or wrong type or opts context is not supported.
func (sch ProductSignScheme) Sign(sk sign.PrivateKey, message []byte, opts *sign.SignatureOpts) []byte {
	var sig []byte
	product := sk.(ProductSigner)
	for i, sch2 := range sch {
		pk := product[i]
		sig2 := sch2.Sign(pk, message, opts)
		sig = append(sig, sig2...)
	}
	return sig
}

// Checks whether the given signature is a valid signature set by
// the private key corresponding to the given public key on the
// given message. opts are additional options which can be nil.
//
// Panics if key is nil or wrong type or opts context is not supported.
func (sch ProductSignScheme) Verify(pk sign.PublicKey, message []byte, sig []byte, opts *sign.SignatureOpts) bool {
	if len(sig) != sch.SignatureSize() {
		return false
	}
	product := pk.(ProductVerifier)
	var offset int
	for i, sch2 := range sch {
		pubKey := product[i]
		start, end := offset, offset+sch2.SignatureSize()
		sig2 := sig[start:end]
		if !sch2.Verify(pubKey, message, sig2, opts) {
			return false
		}
	}
	return true
}

// Deterministically derives a keypair from a seed. If you're unsure,
// you're better off using GenerateKey().
//
// Panics if seed is not of length SeedSize().
func (sch ProductSignScheme) DeriveKey(seed []byte) (sign.PublicKey, sign.PrivateKey) {
	panic("not implemented") // TODO: Implement
}

// Unmarshals a PublicKey from the provided buffer.
func (sch ProductSignScheme) UnmarshalBinaryPublicKey(data []byte) (sign.PublicKey, error) {
	if len(data) != sch.PublicKeySize() {
		return nil, fmt.Errorf("wrong size for public key. HAVE:%d WANT:%d", len(data), sch.PublicKeySize())
	}
	var pubKey ProductVerifier
	var offset int
	for _, sch2 := range sch {
		start, end := offset, offset+sch2.PublicKeySize()
		pubKey2, err := sch2.UnmarshalBinaryPublicKey(data[start:end])
		if err != nil {
			return nil, err
		}
		pubKey = append(pubKey, pubKey2)
		offset += sch2.PublicKeySize()
	}
	return pubKey, nil
}

// Unmarshals a PublicKey from the provided buffer.
func (sch ProductSignScheme) UnmarshalBinaryPrivateKey(data []byte) (sign.PrivateKey, error) {
	if len(data) != sch.PrivateKeySize() {
		return nil, fmt.Errorf("wrong size for private key. HAVE:%d WANT:%d", len(data), sch.PrivateKeySize())
	}
	var privKey ProductSigner
	var offset int
	for _, sch2 := range sch {
		start, end := offset, offset+sch2.PublicKeySize()
		privKey2, err := sch2.UnmarshalBinaryPrivateKey(data[start:end])
		if err != nil {
			return nil, err
		}
		privKey = append(privKey, privKey2)
	}
	return privKey, nil
}

// Size of binary marshalled public keys.
func (sch ProductSignScheme) PublicKeySize() (ret int) {
	for i := range sch {
		ret += sch[i].PublicKeySize()
	}
	return ret
}

// Size of binary marshalled public keys.
func (sch ProductSignScheme) PrivateKeySize() (ret int) {
	for i := range sch {
		ret += sch[i].PrivateKeySize()
	}
	return ret
}

// Size of signatures.
func (sch ProductSignScheme) SignatureSize() (ret int) {
	for i := range sch {
		ret += sch[i].SignatureSize()
	}
	return ret
}

// Size of seeds.
func (sch ProductSignScheme) SeedSize() (ret int) {
	for i := range sch {
		ret += sch[i].SeedSize()
	}
	return ret
}

// Returns whether contexts are supported.
func (sch ProductSignScheme) SupportsContext() bool {
	for i := range sch {
		if !sch[i].SupportsContext() {
			return false
		}
	}
	return true
}

type ProductVerifier []sign.PublicKey

func (pv ProductVerifier) Scheme() sign.Scheme {
	return ProductSignScheme{}
}

func (pv ProductVerifier) Equal(x crypto.PublicKey) bool {
	if pv2, ok := x.(ProductVerifier); ok {
		return slices.EqualFunc(pv, pv2, func(a, b sign.PublicKey) bool {
			return a.Equal(b)
		})
	}
	return false
}

func (pv ProductVerifier) MarshalBinary() ([]byte, error) {
	var ret []byte
	for i := range pv {
		data, err := pv[i].MarshalBinary()
		if err != nil {
			return nil, err
		}
		ret = append(ret, data...)
	}
	return ret, nil
}

type ProductSigner []sign.PrivateKey

func (s ProductSigner) Scheme() sign.Scheme {
	sch := make(ProductSignScheme, len(s))
	for i := range sch {
		sch[i] = s[i].Scheme()
	}
	return ProductSignScheme{}
}

func (s ProductSigner) Public() crypto.PublicKey {
	pub := make(ProductVerifier, len(s))
	for i := range pub {
		pub[i] = s[i].Public().(sign.PublicKey)
	}
	return pub
}

func (s ProductSigner) Sign(r io.Reader, msg []byte, opts crypto.SignerOpts) ([]byte, error) {
	var ret []byte
	for i := range s {
		sig, err := s[i].Sign(r, msg, opts)
		if err != nil {
			return nil, err
		}
		ret = append(ret, sig...)
	}
	return ret, nil
}

func (s ProductSigner) Equal(x crypto.PrivateKey) bool {
	if s2, ok := x.(ProductSigner); ok {
		return slices.EqualFunc(s, s2, func(e1, e2 sign.PrivateKey) bool {
			return e1.Equal(e2)
		})
	}
	return false
}

func (s ProductSigner) MarshalBinary() ([]byte, error) {
	var ret []byte
	for i := range s {
		data, err := s[i].MarshalBinary()
		if err != nil {
			return nil, err
		}
		ret = append(ret, data...)
	}
	return ret, nil
}
