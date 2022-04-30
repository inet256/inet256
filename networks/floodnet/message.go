package floodnet

import (
	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

const purpose = "inet256/floodnet/message"

const (
	modeData      = 0
	modeSolicit   = 1
	modeAdvertise = 2
)

type Message struct {
	Dst     []byte `json:"dst"`
	SrcKey  []byte `json:"src_key"`
	Payload []byte `json:"payload"`
	Mode    uint8  `json:"mode"`

	Hops int `json:"hops"`
}

func newMessage(pk p2p.PrivateKey, dst inet256.Addr, data []byte, hops int, mode uint8) Message {
	return Message{
		Dst:     dst[:],
		SrcKey:  inet256.MarshalPublicKey(pk.Public()),
		Payload: data,
		Mode:    mode,

		Hops: hops,
	}
}

func (m Message) GetSrc() (inet256.Addr, error) {
	pubKey, err := inet256.ParsePublicKey(m.SrcKey)
	if err != nil {
		return inet256.Addr{}, err
	}
	return inet256.NewAddr(pubKey), nil
}

func (m Message) GetDst() inet256.Addr {
	dst := inet256.Addr{}
	copy(dst[:], m.Dst)
	return dst
}
