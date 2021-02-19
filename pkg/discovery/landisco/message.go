package landisco

import (
	"crypto/hmac"
	"math/rand"

	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/sha3"
)

type Advertisement struct {
	Transports []string `json:"transports"`
}

const MinMessageSize = 32 + 32

type Message []byte

func NewMessage(id p2p.PeerID, payload []byte) Message {
	ciph, err := chacha20poly1305.NewX(id[:])
	if err != nil {
		panic(err)
	}
	nonce := [32]byte{}
	if n, err := rand.Read(nonce[:]); err != nil || n < len(nonce) {
		panic(err)
	}

	preimage := [64]byte{}
	copy(preimage[:32], id[:])
	copy(preimage[32:], nonce[:])
	sum := sha3.Sum256(preimage[:])

	m := make(Message, MinMessageSize)
	m.setNonce(nonce[:])
	m.setMAC(sum[:])
	m = ciph.Seal(m, nonce[:ciph.NonceSize()], payload, nil)

	return m
}

func ParseMessage(x []byte) (Message, error) {
	if len(x) < MinMessageSize {
		return nil, errors.Errorf("buffer too short to be message")
	}
	return Message(x), nil
}

func (m Message) GetMAC() [32]byte {
	return copy32(m[:32])
}

func (m Message) setMAC(x []byte) {
	if len(x) < 32 {
		panic("set mac with len < 32")
	}
	copy(m[:32], x)
}

func (m Message) GetNonce() [32]byte {
	return copy32(m[32:64])
}

func (m Message) setNonce(x []byte) {
	if len(x) < 32 {
		panic("set nonce with len < 32")
	}
	copy(m[32:64], x)
}

func (m Message) GetCiphertext() []byte {
	return m[64:]
}

func copy32(x []byte) [32]byte {
	y := [32]byte{}
	copy(y[:], x)
	return y
}

func UnpackMessage(m Message, ids []p2p.PeerID) (int, []byte, error) {
	mac := m.GetMAC()
	nonce := m.GetNonce()
	preimage := [64]byte{}
	copy(preimage[32:], nonce[:])
	for i, id := range ids {
		copy(preimage[:32], id[:])
		sum := sha3.Sum256(preimage[:])
		// I don't know where we would ever leak this timing information, but whatever.
		if hmac.Equal(sum[:], mac[:]) {
			ciph, err := chacha20poly1305.NewX(id[:])
			if err != nil {
				panic(err)
			}
			ptext, err := ciph.Open(nil, nonce[:ciph.NonceSize()], m.GetCiphertext(), nil)
			if err != nil {
				return -1, nil, err
			}
			return i, ptext, nil
		}
	}
	return -1, nil, errors.Errorf("no provided peer sent this message")
}
