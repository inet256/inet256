package inet256ipc

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/inet256/inet256/pkg/inet256"
)

type MessageType uint32

const (
	MT_Data      = MessageType('D'<<24 | 'A'<<16 | 'T'<<8 | 'A')
	MT_PublicKey = MessageType('P'<<24 | 'U'<<16 | 'B'<<8 | 'K')
	MT_FindAddr  = MessageType('F'<<24 | 'I'<<16 | 'N'<<8 | 'D')
	MT_KeepAlive = MessageType('K'<<24 | 'E'<<16 | 'E'<<8 | 'P')
)

const (
	MaxMessageLen = 4 + 32 + inet256.MTU
	MinMessageLen = 4
	ReqIDLen      = 16

	MinDataMsgLen = MinMessageLen + 32
	MinAskMsgLen  = MinMessageLen + ReqIDLen
)

type Message []byte

func AsMessage(x []byte, isOutgoing bool) (Message, error) {
	if len(x) < MinMessageLen {
		return nil, fmt.Errorf("too short to be message len=%d", len(x))
	}
	m := Message(x)
	switch m.GetType() {
	case MT_Data:
		if len(x) < MinDataMsgLen {
			return nil, fmt.Errorf("too short to be Data message")
		}
	case MT_FindAddr, MT_PublicKey:
		if len(x) < MinAskMsgLen {
			return nil, fmt.Errorf("message type=%v does not contain request-id", m.GetType())
		}
	case MT_KeepAlive:
	default:
		return nil, fmt.Errorf("unrecognized type %v", m.GetType())
	}
	return m, nil
}

func (m Message) SetType(mt MessageType) {
	binary.BigEndian.PutUint32(m[:4], uint32(mt))
}

func (m Message) GetType() MessageType {
	return MessageType(binary.BigEndian.Uint32(m[:4]))
}

func (m Message) IsTell() bool {
	switch m.GetType() {
	case MT_Data, MT_KeepAlive:
		return true
	default:
		return false
	}
}

func (m Message) IsAsk() bool {
	switch m.GetType() {
	case MT_FindAddr, MT_PublicKey:
		return true
	default:
		return false
	}
}

func (m Message) DataAddrBytes() []byte {
	return m[4 : 4+32]
}

func (m Message) DataPayload() []byte {
	return m[4+32:]
}

func (m Message) DataMsg() DataMsg {
	return DataMsg{
		Addr:    inet256.AddrFromBytes(m[4 : 4+32]),
		Payload: m[4+32:],
	}
}

func (m Message) GetRequestID() (ret [16]byte) {
	return *(*[16]byte)(m[4 : 4+16])
}

func (m Message) SetRequestID(id [16]byte) {
	copy(m[4:4+16], id[:])
}

func (m Message) AskBody() []byte {
	return m[4+16:]
}

func (m Message) LookupPublicKeyReq() (*LookupPublicKeyReq, error) {
	return parseJSON[LookupPublicKeyReq](m[4+16:])
}

func (m Message) LookupPublicKeyRes() (*LookupPublicKeyRes, error) {
	res, err := parseJSON[Response[LookupPublicKeyRes]](m[4+16:])
	if err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return &res.Success, nil
}

func (m Message) FindAddrReq() (*FindAddrReq, error) {
	return parseJSON[FindAddrReq](m[4+16:])
}

func (m Message) FindAddrRes() (*FindAddrRes, error) {
	res, err := parseJSON[Response[FindAddrRes]](m[4+16:])
	if err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return &res.Success, nil
}

func (m Message) MTUReq() (*MTUReq, error) {
	return parseJSON[MTUReq](m[4+16:])
}

func (m Message) MTURes() (*MTURes, error) {
	return parseJSON[MTURes](m[4+16:])
}

// WriteDataMessage places a message in the frame
func WriteDataMessage(dst []byte, addr inet256.Addr, data []byte) int {
	l := 4 + 32 + len(data)
	m := Message(dst[:l])
	m.SetType(MT_Data)
	copyExact(m.DataAddrBytes(), addr[:])
	copyExact(m.DataPayload(), data)
	return l
}

func WriteKeepAlive(dst []byte) int {
	l := 4
	m := Message(dst[:l])
	m.SetType(MT_KeepAlive)
	return l
}

func WriteAskMessage(dst []byte, reqID [16]byte, mtype MessageType, x any) int {
	data, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	l := 4 + 16 + len(data)
	m := Message(dst[:l])
	m.SetType(mtype)
	m.SetRequestID(reqID)
	copyExact(m.AskBody(), data)
	return l
}

func WriteRequest(dst []byte, reqID [16]byte, mtype MessageType, x any) int {
	return WriteAskMessage(dst, reqID, mtype, x)
}

func WriteSuccess[T any](dst []byte, reqID [16]byte, mtype MessageType, x T) int {
	return WriteAskMessage(dst, reqID, mtype, Response[T]{Success: x})
}

func WriteError[T any](dst []byte, reqID [16]byte, mtype MessageType, err error) int {
	return WriteAskMessage(dst, reqID, mtype, Response[T]{Error: err.Error()})
}

func NewRequestID() (ret [16]byte) {
	io.ReadFull(rand.Reader, ret[:])
	return ret
}

func parseJSON[T any](x []byte) (*T, error) {
	var y T
	if err := json.Unmarshal(x, &y); err != nil {
		return nil, err
	}
	return &y, nil
}

type DataMsg struct {
	Addr    inet256.Addr
	Payload []byte
}

type Response[T any] struct {
	Success T      `json:"ok,omitempty"`
	Error   string `json:"error,omitempty"`
}

type FindAddrReq struct {
	Prefix []byte `json:"prefix"`
	Nbits  int    `json:"nbits"`
}
type FindAddrRes struct {
	Addr inet256.Addr `json:"addr,omitempty"`
}

type LookupPublicKeyReq struct {
	Target inet256.Addr `json:"target"`
}
type LookupPublicKeyRes struct {
	PublicKey []byte `json:"public_key,omitempty"`
}

type MTUReq struct {
	Target inet256.Addr `json:"target"`
}
type MTURes struct {
	MTU int `json:"mtu"`
}

func copyExact(dst, src []byte) {
	n := copy(dst, src)
	if n != len(dst) {
		panic(len(dst))
	}
	if n != len(src) {
		panic(len(src))
	}
}
