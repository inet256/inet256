package serde

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"

	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"

	"github.com/inet256/inet256/pkg/inet256"
)

func MarshalPrivateKey(privKey inet256.PrivateKey) []byte {
	data, err := x509.MarshalPKCS8PrivateKey(privKey.BuiltIn())
	if err != nil {
		panic(err)
	}
	return data
}

func ParsePrivateKey(data []byte) (inet256.PrivateKey, error) {
	privKey, err := x509.ParsePKCS8PrivateKey(data)
	if err != nil {
		return nil, err
	}
	return inet256.PrivateKeyFromBuiltIn(privKey.(crypto.Signer))
}

func MarshalPrivateKeyPEM(privateKey inet256.PrivateKey) ([]byte, error) {
	data := MarshalPrivateKey(privateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: data,
	})
	return privKeyPEM, nil
}

func ParsePrivateKeyPEM(data []byte) (inet256.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("key file does not contain PEM")
	}
	if block.Type != "PRIVATE KEY" {
		return nil, errors.New("wrong type for PEM block")
	}
	return ParsePrivateKey(block.Bytes)
}

func MarshalAddrs[T p2p.Addr](xs []T) []string {
	ys := make([]string, len(xs))
	for i := range xs {
		data, err := xs[i].MarshalText()
		if err != nil {
			panic(err)
		}
		ys[i] = string(data)
	}
	return ys
}

func ParseAddrs[T p2p.Addr](parser p2p.AddrParser[T], xs []string) ([]T, error) {
	ys := make([]T, len(xs))
	for i := range xs {
		addr, err := parser([]byte(xs[i]))
		if err != nil {
			return nil, err
		}
		ys[i] = addr
	}
	return ys, nil
}
