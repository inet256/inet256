package landisco

import (
	"github.com/pkg/errors"
	"go.brendoncarroll.net/tai64"
	"go.inet256.org/inet256/pkg/inet256"
	"golang.org/x/crypto/chacha20poly1305"
)

type Advertisement struct {
	Transports []string `json:"transports"`
	Solicit    bool     `json:"solicit"`
}

const MinMessageSize = 12 + 16

type Message []byte

func NewMessage(now tai64.TAI64N, id inet256.Addr, payload []byte) Message {
	ciph, err := chacha20poly1305.New(id[:])
	if err != nil {
		panic(err)
	}
	nonce := [12]byte{}
	copy(nonce[:], now.Marshal())

	m := make(Message, 0)
	m = append(m, nonce[:]...)
	m = ciph.Seal(m, nonce[:], payload, nil)

	return m
}

func ParseMessage(x []byte) (Message, error) {
	if len(x) < MinMessageSize {
		return nil, errors.Errorf("buffer too short to be message")
	}
	return Message(x), nil
}

func (m Message) GetTAI64N() (tai64.TAI64N, error) {
	return tai64.ParseN(m[0:12])
}

func (m Message) Open(ids []inet256.Addr) (int, []byte, error) {
	nonce := m[0:12]
	ctext := m[12:]
	for i, id := range ids {
		ciph, err := chacha20poly1305.New(id[:])
		if err != nil {
			panic(err)
		}
		ptext, err := ciph.Open(nil, nonce, ctext, nil)
		if err != nil {
			continue
		}
		return i, ptext, nil
	}
	return -1, nil, errors.Errorf("no provided peer sent this message")
}
