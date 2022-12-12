package inet256

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/sha3"
)

const (
	// MaxPublicKeySize is the maximum size of a serialized PublicKey in bytes
	MaxPublicKeySize = 1 << 15
	// MaxSignatureSize is the maximum size of a signature in bytes
	MaxSignatureSize = 1 << 15
)

// PublicKey is a signing key used to prove identity within INET256.
type PublicKey interface {
	// BuiltIn returns a crypto.PublicKey from the standard library
	BuiltIn() crypto.PublicKey

	isPublicKey()
}

// PublicKeyFromBuiltIn returns a PublicKey from the BuiltIn crypto.PublicKey type
func PublicKeyFromBuiltIn(x crypto.PublicKey) (PublicKey, error) {
	switch x := x.(type) {
	case ed25519.PublicKey:
		return (*Ed25519PublicKey)(x), nil
	case *rsa.PublicKey:
		if x.N.BitLen() < 1024 {
			return nil, errors.New("rsa keys must be >= 1024 bits for use in INET256")
		}
		return (*RSAPublicKey)(x), nil
	default:
		return nil, fmt.Errorf("public keys of type %T are not supported by INET256", x)
	}
}

// Ed25519PublicKey implements PublicKey
type Ed25519PublicKey [ed25519.PublicKeySize]byte

// BuiltIn implements PublicKey.BuiltIn
func (pk *Ed25519PublicKey) BuiltIn() crypto.PublicKey {
	return ed25519.PublicKey(pk[:])
}

func (pk *Ed25519PublicKey) isPublicKey() {}

// RSAPublicKey implements PublicKey
type RSAPublicKey rsa.PublicKey

// BuiltIn implements PublicKey.BuiltIn
func (pk *RSAPublicKey) BuiltIn() crypto.PublicKey {
	return (*rsa.PublicKey)(pk)
}

func (pk *RSAPublicKey) isPublicKey() {}

// ParsePublicKey attempts to parse a PublicKey from data
func ParsePublicKey(data []byte) (PublicKey, error) {
	pubKey, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, err
	}
	return PublicKeyFromBuiltIn(pubKey)
}

// MarshalPublicKey marshals pubKey and the resulting bytes to out.
// All keys returned by ParsePublic key will successfully marshal, so a panic indicates a bug.
func MarshalPublicKey(out []byte, pubKey PublicKey) []byte {
	data, err := x509.MarshalPKIXPublicKey(pubKey.BuiltIn())
	if err != nil {
		panic(err)
	}
	return append(out, data...)
}

type PrivateKey interface {
	// BuiltIn returns a crypto.Signer from the standard library.
	BuiltIn() crypto.Signer

	// Public returns the corresponding public key for this PrivateKey
	Public() PublicKey

	isPrivateKey()
}

// PrivateKeyFromBuiltIn
func PrivateKeyFromBuiltIn(x crypto.Signer) (PrivateKey, error) {
	switch x := x.(type) {
	case ed25519.PrivateKey:
		return (*Ed25519PrivateKey)(x), nil
	case *rsa.PrivateKey:
		if x.N.BitLen() < 1024 {
			return nil, errors.New("rsa keys must be >= 1024 bits for use in INET256")
		}
		return (*RSAPrivateKey)(x), nil
	default:
		return nil, fmt.Errorf("private keys of type %T are not supported by INET256", x)
	}
}

type Ed25519PrivateKey [ed25519.PrivateKeySize]byte

func (pk *Ed25519PrivateKey) BuiltIn() crypto.Signer {
	return ed25519.PrivateKey(pk[:])
}

func (pk *Ed25519PrivateKey) Public() PublicKey {
	pub, err := PublicKeyFromBuiltIn(pk.BuiltIn().Public())
	if err != nil {
		panic(err)
	}
	return pub
}

func (pk *Ed25519PrivateKey) isPrivateKey() {}

type RSAPrivateKey rsa.PrivateKey

func (pk *RSAPrivateKey) BuiltIn() crypto.Signer {
	return (*rsa.PrivateKey)(pk)
}

func (pk *RSAPrivateKey) Public() PublicKey {
	pub, err := PublicKeyFromBuiltIn(pk.BuiltIn().Public())
	if err != nil {
		panic(err)
	}
	return pub
}

func (pk *RSAPrivateKey) isPrivateKey() {}

// GenerateKey generates a new key pair using entropy read from rng.
//
// The algorithm used currently is Ed25519, but this may change at any time, and callers *must not* depend on this.
// If a specific public key algorithm is required use the standard library to generate a key and convert it using PrivateKeyFromBuiltIn.
func GenerateKey(rng io.Reader) (PublicKey, PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rng)
	if err != nil {
		return nil, nil, err
	}
	pub2, err := PublicKeyFromBuiltIn(pub)
	if err != nil {
		return nil, nil, err
	}
	priv2, err := PrivateKeyFromBuiltIn(priv)
	if err != nil {
		return nil, nil, err
	}
	return pub2, priv2, nil
}

// Sign appends a signature to out and returns it.
// Sign implements the INET256 Signature Scheme.
func Sign(out []byte, purpose string, privateKey PrivateKey, msg []byte) []byte {
	input := prehash(purpose, msg)
	switch priv := privateKey.(type) {
	case *Ed25519PrivateKey:
		sig := ed25519.Sign(priv[:], input[:])
		out = append(out, sig...)
	case *RSAPrivateKey:
		sig, err := rsa.SignPKCS1v15(nil, (*rsa.PrivateKey)(priv), crypto.Hash(0), input[:])
		if err != nil {
			panic(err)
		}
		out = append(out, sig...)
	default:
		panic(privateKey)
	}
	return out
}

// Verify checks that sig is a valid signature for msg, produces by publicKey.
// Verify returns true for a correct signature and false otherwise.
// Verify implements the INET256 Signature Scheme
func Verify(purpose string, publicKey PublicKey, msg, sig []byte) bool {
	input := prehash(purpose, msg)
	switch pub := publicKey.(type) {
	case *Ed25519PublicKey:
		return ed25519.Verify(pub[:], input[:], sig)
	case *RSAPublicKey:
		err := rsa.VerifyPKCS1v15((*rsa.PublicKey)(pub), crypto.Hash(0), input[:], sig)
		return err == nil
	default:
		panic(publicKey)
	}
}

func prehash(purpose string, data []byte) (ret [64]byte) {
	h := sha3.NewCShake256(nil, []byte(purpose))
	h.Write(data)
	h.Read(ret[:])
	return ret
}
