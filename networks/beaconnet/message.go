package beaconnet

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"go.inet256.org/inet256/src/inet256"
)

const (
	TypeData   = uint8(0)
	TypeBeacon = uint8(1)
)

var broadcastAddr = inet256.Addr{}

const (
	HeaderSize = 1 + 32 + 32

	srcStart = 1
	srcEnd   = 1 + 32
	dstStart = srcEnd
	dstEnd   = dstStart + 32
)

type Header [HeaderSize]byte

func ParseMessage(data []byte) (Header, []byte, error) {
	if len(data) < HeaderSize {
		return Header{}, nil, errors.Errorf("not long enough to contain header")
	}
	hdr := Header{}
	copy(hdr[:], data[:HeaderSize])
	var body []byte
	if len(data) > HeaderSize {
		body = data[HeaderSize:]
	}
	return hdr, body, nil
}

func (h *Header) GetType() uint8 {
	return h[0]
}

func (h *Header) SetType(x uint8) {
	h[0] = x
}

func (h *Header) GetSrc() inet256.Addr {
	return inet256.IDFromBytes(h[srcStart:srcEnd])
}

func (h *Header) SetSrc(x inet256.Addr) {
	copy(h[srcStart:srcEnd], x[:])
}

func (h *Header) GetDst() inet256.Addr {
	return inet256.IDFromBytes(h[dstStart:dstEnd])
}

func (h *Header) SetDst(x inet256.Addr) {
	copy(h[dstStart:dstEnd], x[:])
}

func (h *Header) String() string {
	return fmt.Sprintf("Header{%v, %v -> %v}", h.GetType(), h.GetSrc(), h.GetDst())
}

var sigPurpose = inet256.SigCtxString("inet256/beaconnet/beacon")

type Beacon struct {
	PublicKey []byte `json:"public_key"`
	Counter   uint64 `json:"counter"`
	Sig       []byte `json:"sig"`
}

func newBeacon(pki *inet256.PKI, privateKey inet256.PrivateKey, now time.Time) *Beacon {
	now = now.UTC()
	counter := uint64(now.UnixNano())
	counterBytes := [8]byte{}
	binary.BigEndian.PutUint64(counterBytes[:], counter)
	sig := pki.Sign(&sigPurpose, privateKey, counterBytes[:], nil)
	return &Beacon{
		PublicKey: inet256.MarshalPublicKey(nil, privateKey.Public().(inet256.PublicKey)),
		Counter:   counter,
		Sig:       sig,
	}
}

func verifyBeacon(pki *inet256.PKI, b Beacon) (inet256.PublicKey, error) {
	pubKey, err := inet256.ParsePublicKey(b.PublicKey)
	if err != nil {
		return nil, err
	}
	counterBytes := [8]byte{}
	binary.BigEndian.PutUint64(counterBytes[:], b.Counter)
	if !pki.Verify(&sigPurpose, pubKey, counterBytes[:], b.Sig) {
		return nil, errors.New("verifyBeacon: invalid signature")
	}
	return pubKey, nil
}

func jsonMarshal(x interface{}) []byte {
	data, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	return data
}
