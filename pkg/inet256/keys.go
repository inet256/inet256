package inet256

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
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
