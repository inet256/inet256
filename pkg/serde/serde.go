package serde

import (
	"crypto/x509"
	"encoding/pem"

	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"
)

func MarshalPrivateKey(privKey p2p.PrivateKey) []byte {
	data, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		panic(err)
	}
	return data
}

func ParsePrivateKey(data []byte) (p2p.PrivateKey, error) {
	privKey, err := x509.ParsePKCS8PrivateKey(data)
	if err != nil {
		return nil, err
	}
	privKey2, ok := privKey.(p2p.PrivateKey)
	if !ok {
		return nil, errors.Errorf("unsupported private key type")
	}
	return privKey2, nil
}

func MarshalPrivateKeyPEM(privateKey p2p.PrivateKey) ([]byte, error) {
	data := MarshalPrivateKey(privateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: data,
	})
	return privKeyPEM, nil
}

func ParsePrivateKeyPEM(data []byte) (p2p.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("key file does not contain PEM")
	}
	if block.Type != "PRIVATE KEY" {
		return nil, errors.New("wrong type for PEM block")
	}
	return ParsePrivateKey(block.Bytes)
}

func MarshalAddrs(xs []p2p.Addr) []string {
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

type AddrParserFunc = func([]byte) (p2p.Addr, error)

func ParseAddrs(parser AddrParserFunc, xs []string) ([]p2p.Addr, error) {
	ys := make([]p2p.Addr, len(xs))
	for i := range xs {
		addr, err := parser([]byte(xs[i]))
		if err != nil {
			return nil, err
		}
		ys[i] = addr
	}
	return ys, nil
}
