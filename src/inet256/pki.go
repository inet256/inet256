package inet256

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudflare/circl/sign"
	dilithium2 "github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/cloudflare/circl/sign/ed25519"
	inet256crypto "go.inet256.org/inet256/src/internal/crypto"
)

type PublicKey sign.PublicKey

type PrivateKey sign.PrivateKey

// PublicFromPrivate returns a PublicKey from the PrivateKey
// This paves over a decision in CIRCL to favor compatibility with the standard library over ease of use.
func PublicFromPrivate(x PrivateKey) PublicKey {
	return x.Public().(PublicKey)
}

// SigCtx represents a context for producing a signature.
// It is used to allow a single key to be used to sign multiple formats.
type SigCtx inet256crypto.ME256

// SigCtxString creates a new signature context from a string.
// You should have one of these for every type of data that you sign in your app.
// You only need to call it once at startup and then you can reuse the SigCtx.
func SigCtxString(x string) SigCtx {
	return SigCtx(inet256crypto.Sum256(nil, []byte(x)))
}

var DefaultPKI = PKI{
	Default: SignAlgo_Ed25519,
	Schemes: map[string]sign.Scheme{
		SignAlgo_Ed25519:    ed25519.Scheme(),
		SignAlgo_Dilithium2: dilithium2.Scheme(),
		SignAlgo_Product(SignAlgo_Ed25519, SignAlgo_Dilithium2): SignScheme_Product(ed25519.Scheme(), dilithium2.Scheme()),
	},
}

// GenerateKey calls GenerateKey on the DefaultPKI
func GenerateKey() (PublicKey, PrivateKey, error) {
	return DefaultPKI.GenerateKey()
}

// MarshalPublicKey calls MarshalPublicKey on the DefaultPKI
func MarshalPublicKey(out []byte, pubKey PublicKey) []byte {
	out, err := DefaultPKI.MarshalPublicKey(out, pubKey)
	if err != nil {
		panic(err)
	}
	return out
}

// ParsePublicKey calls ParsePublicKey on the default PKI
func ParsePublicKey(x []byte) (PublicKey, error) {
	return DefaultPKI.ParsePublicKey(x)
}

// ParsePrivateKey callse ParsePrivateKEy on the default PKI
func ParsePrivateKey(x []byte) (PrivateKey, error) {
	return DefaultPKI.ParsePrivateKey(x)
}

func MarshalPrivateKey(out []byte, privKey PrivateKey) []byte {
	data, err := DefaultPKI.MarshalPrivateKey(out, privKey)
	if err != nil {
		panic(err)
	}
	return data
}

// PKI is a public key infrastructure.
// It includes a set of named signing schemes.
type PKI struct {
	Default string
	Schemes map[string]sign.Scheme
}

func (pki *PKI) GenerateKey() (PublicKey, PrivateKey, error) {
	sch := pki.Schemes[pki.Default]
	if sch == nil {
		return nil, nil, fmt.Errorf("no default scheme set")
	}
	return sch.GenerateKey()
}

// ID uniquely identifies an INET256 node.
func (pki *PKI) NewID(pubKey PublicKey) ID {
	data, err := pki.MarshalPublicKey(nil, pubKey)
	if err != nil {
		panic(err)
	}
	return ID(inet256crypto.Sum256(nil, data))
}

func (pki *PKI) ParsePublicKey(data []byte) (PublicKey, error) {
	tag, data, err := readTag(data)
	if err != nil {
		return nil, err
	}
	sch, found := pki.Schemes[tag]
	if !found {
		return nil, fmt.Errorf("unknown algorithm %v", tag)
	}
	return sch.UnmarshalBinaryPublicKey(data)
}

// MarshalPublicKey appends a public key to out
func (pki *PKI) MarshalPublicKey(out []byte, pubKey PublicKey) ([]byte, error) {
	tag, err := pki.findTag(pubKey.Scheme())
	if err != nil {
		return nil, err
	}
	out = writeTag(out, tag)
	pubKeyBytes, err := pubKey.MarshalBinary()
	if err != nil {
		return nil, err
	}
	out = append(out, pubKeyBytes...)
	return out, nil
}

func (pki *PKI) ParsePrivateKey(data []byte) (PrivateKey, error) {
	tag, data, err := readTag(data)
	if err != nil {
		return nil, err
	}
	tag, isPrivate := strings.CutSuffix(tag, PrivateSuffix)
	if !isPrivate {
		return nil, fmt.Errorf("not a private key. type=%v", tag)
	}
	sch := pki.Schemes[tag]
	return sch.UnmarshalBinaryPrivateKey(data)
}

func (pki *PKI) MarshalPrivateKey(out []byte, privateKey PrivateKey) ([]byte, error) {
	tag, err := pki.findTag(privateKey.Scheme())
	if err != nil {
		return nil, err
	}
	tag += PrivateSuffix
	out = writeTag(out, tag)
	keyBytes, err := privateKey.MarshalBinary()
	if err != nil {
		return nil, err
	}
	out = append(out, keyBytes...)
	return out, nil
}

// Sign appends a signature to out (which may be nil) and returns it.
func (pki *PKI) Sign(sigCtx *SigCtx, privateKey PrivateKey, msg []byte, out []byte) []byte {
	if sigCtx == nil {
		panic("sigCtx cannot be nil")
	}
	// input is the input to the signature algorithm.
	var input [512]byte
	inet256crypto.XOF((*inet256crypto.ME256)(sigCtx), msg, input[:])

	sig := privateKey.Scheme().Sign(privateKey, input[:], nil)
	return append(out, sig...)
}

func (pki *PKI) Verify(sigCtx *SigCtx, pubKey PublicKey, msg []byte, sig []byte) bool {
	if sigCtx == nil {
		panic("sigCtx cannot be nil")
	}
	// input is the input to the signature algorithm.
	var input [512]byte
	inet256crypto.XOF((*inet256crypto.ME256)(sigCtx), msg, input[:])
	return pubKey.Scheme().Verify(pubKey, input[:], sig, nil)
}

func (pki *PKI) findTag(target sign.Scheme) (string, error) {
	var matchingSchemes []string
	for tag, sch := range pki.Schemes {
		if reflect.DeepEqual(sch, target) {
			matchingSchemes = append(matchingSchemes, tag)
		}
	}
	switch len(matchingSchemes) {
	case 0:
		return "", fmt.Errorf("no scheme found for %v :: %T", target, target)
	case 1:
	default:
		return "", fmt.Errorf("multiple schemes found for %v :: %T", target, target)
	}
	return matchingSchemes[0], nil
}

func writeTag(out []byte, tag string) []byte {
	out = binary.LittleEndian.AppendUint16(out, uint16(len(tag)))
	out = append(out, tag...)
	return out
}

func readTag(data []byte) (string, []byte, error) {
	// read the tagLen
	if len(data) < 2 {
		return "", nil, fmt.Errorf("too short to be INET256 key")
	}
	tagLen := binary.LittleEndian.Uint16(data[:2])
	data = data[2:]
	// read the tag
	if len(data) < int(tagLen) {
		return "", nil, fmt.Errorf("too short to contain type tag of len %d", tagLen)
	}
	tag := string(data[:tagLen])
	data = data[tagLen:]

	return tag, data, nil
}

// Everyone returns an ID which is canonically used to represent the set of all IDs in applications
// It has 1s for every bit.
func Everyone() (ret ID) {
	for i := range ret {
		ret[i] = 0xff
	}
	return ret
}

// Noone returns an INET256 ID which is canonically used to represent the empty set of IDs in applications
// It has 0s for every bit.
func Noone() ID {
	return ID{}
}

const (
	SignAlgo_Ed25519    = "ed25519"
	SignAlgo_Dilithium2 = "dilithium2"
)

func SignAlgo_Product(tags ...string) string {
	tags = append([]string{"product"}, tags...)
	return "(" + strings.Join(tags, " ") + ")"
}

const PrivateSuffix = ".private"

func SignScheme_Ed25519() sign.Scheme {
	return ed25519.Scheme()
}

func SignScheme_Dilithium2() sign.Scheme {
	return dilithium2.Scheme()
}

func SignScheme_Product(schs ...sign.Scheme) sign.Scheme {
	return inet256crypto.ProductSignScheme(schs)
}
