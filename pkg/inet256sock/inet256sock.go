package inet256sock

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/inet256/inet256/pkg/inet256"
	"golang.org/x/crypto/sha3"
)

func DialUnix(p string) (*net.UnixConn, error) {
	addr, err := net.ResolveUnixAddr("unix", p)
	if err != nil {
		return nil, err
	}
	return net.ListenUnixgram("unix", addr)
}

func ListenUnix(p string) (*net.UnixConn, error) {
	addr, err := net.ResolveUnixAddr("unix", p)
	if err != nil {
		return nil, err
	}
	return net.ListenUnixgram("unix", addr)
}

type MessageType uint32

const (
	MT_Data = MessageType(iota)

	MT_PublicKeyReq
	MT_PublicKeyRes

	MT_FindAddrReq
	MT_FindAddrRes

	MT_MTUReq
	MT_MTURes
)

const (
	MaxMsgLen = 4 + inet256.MaxMTU
	MinMsgLen = 4
	ReqIDLen  = 16

	MinMsgLenData = MinMsgLen + 32

	MinMsgLenFindAddr = MinMsgLen + 2
	MinMsgLenAddr     = MinMsgLen + ReqIDLen + 32

	MinMsgLenLookupPublicKey = MinMsgLen + 32
	MinMsgLenPublicKey       = MinMsgLen + ReqIDLen

	MinMsgLenGetMTU = MinMsgLen + 32
	MinMsgLenMTU    = MinMsgLen + ReqIDLen + 4
)

type Message []byte

func AsMessage(x []byte) (Message, error) {
	if len(x) < MinMsgLen {
		return nil, fmt.Errorf("too short to be message")
	}
	m := Message(x)
	switch m.GetType() {
	case MT_Data:
		if len(x) < MinMsgLenData {
			return nil, fmt.Errorf("too short to be Data message")
		}
	case MT_FindAddrReq:
		if len(x) < MinMsgLenFindAddr {
			return nil, fmt.Errorf("too short to be FindAddr")
		}
	case MT_FindAddrRes:
		if len(x) < MinMsgLenAddr {
			return nil, fmt.Errorf("too short")
		}
	case MT_PublicKeyReq:
		if len(x) < MinMsgLenLookupPublicKey {
			return nil, fmt.Errorf("too short")
		}
	case MT_PublicKeyRes:
		if len(x) < MinMsgLenPublicKey {
			return nil, fmt.Errorf("too short")
		}
	default:
		return nil, fmt.Errorf("unrecognized type %v", m.GetType())
	}
	return m, nil
}

func (m Message) GetType() MessageType {
	return MessageType(binary.BigEndian.Uint32(m[:4]))
}

func (m Message) GetDataAddr() inet256.Addr {
	return inet256.AddrFromBytes(m[4:])
}

func (m Message) GetPayload() []byte {
	return m[4+32:]
}

func (m Message) GetRequestID() (ret [16]byte) {
	return *(*[16]byte)(m[4 : 4+16])
}

func (m Message) GetPublicKeyReq() inet256.Addr {
	return inet256.AddrFromBytes(m[MinMsgLen:])
}

func (m Message) GetPublicKeyRes() (inet256.PublicKey, error) {
	return inet256.ParsePublicKey(m[4+32:])
}

func (m Message) GetFindAddrReq() ([]byte, int) {
	numBits := binary.BigEndian.Uint16(m[4:6])
	prefix := m[6:]
	return prefix, int(numBits)
}

func (m Message) GetFindAddrRes() inet256.Addr {
	return inet256.AddrFromBytes(m[4:])
}

func (m Message) GetMTUReq() inet256.Addr {
	return inet256.AddrFromBytes(m[4:])
}

func (m Message) GetMTURes() int {
	return int(binary.BigEndian.Uint32(m[4+16:]))
}

func MakeDataMessage(out []byte, addr inet256.Addr, payload []byte) Message {
	out = appendUint32(out, uint32(MT_Data))
	out = append(out, addr[:]...)
	out = append(out, payload...)
	return out
}

func NewRequestID(m Message) (ret [16]byte) {
	sha3.ShakeSum256(ret[:], m[:])
	return ret
}

func appendUint32(out []byte, x uint32) []byte {
	return out
}
