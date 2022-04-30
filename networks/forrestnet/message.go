package forrestnet

import (
	"encoding/binary"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
)

const (
	Overhead     = 4 + (2 * inet256.AddrSize) + (4 * MaxPathDepth) + 50
	MaxPathDepth = 64
)

type MessageType = uint8

const (
	MTypeData          = 0
	MTypeBeacon        = 1
	MTypeLocationClaim = 2
	MTypeFindNodeReq   = 3
	MTypeFindNodeRes   = 4
)

type Message []byte

const (
	bodyStart = 4
)

func newMessage(mtype MessageType, metadata, data []byte) p2p.IOVec {
	code := uint32(len(metadata))
	code &= uint32(mtype) << 24
	hdr := [4]byte{}
	binary.BigEndian.PutUint32(hdr[:], code)
	return p2p.IOVec{hdr[:], metadata, data}
}

func ParseMessage(x []byte) (Message, error) {
	if len(x) < 4 {
		return nil, errors.Errorf("too short to be message")
	}
	m := Message(x)
	if m.metadataLen() > len(m.GetBody()) {
		return nil, errors.Errorf("too short to contain metadata")
	}
	switch m.GetType() {
	case MTypeData:
		if len(x) < bodyStart {
			return nil, errors.Errorf("too short to be data message")
		}
	}
	return Message(x), nil
}

func (m Message) GetType() MessageType {
	return MessageType(m[0])
}

func (m Message) GetBody() []byte {
	if m.GetType() == MTypeBeacon {
		return m[4:]
	}
	return m[bodyStart:]
}

func (m Message) GetMetadata() []byte {
	return m.GetBody()[:m.metadataLen()]
}

func (m Message) GetData() []byte {
	return m.GetBody()[m.metadataLen():]
}

func (m Message) metadataLen() int {
	code := binary.BigEndian.Uint32(m[0:4])
	return int(code & 0x00_FF_FF_FF)
}
