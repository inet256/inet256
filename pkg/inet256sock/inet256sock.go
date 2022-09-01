package inet256sock

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/inet256/inet256/pkg/inet256"
	"golang.org/x/sync/errgroup"
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

// ServeNode reads and writes packets from a DGRAM unix conn, using the node to
// send network data and answer requests.
func ServeNode(ctx context.Context, node inet256.Node, pconn *net.UnixConn) error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		defer pconn.Close()
		<-ctx.Done()
		return ctx.Err()
	})
	eg.Go(func() error {
		buf := make([]byte, MaxMsgLen)
		for {
			n, _, err := pconn.ReadFromUnix(buf)
			if err != nil {
				return err
			}
			msg, err := AsMessage(buf[:n])
			if err != nil {
				logrus.Warnf("leaving")
				continue
			}
		}
	})
	eg.Go(func() error {
		buf := make([]byte, 32+inet256.MaxMTU)
		for {
			var sendErr error
			if err := node.Receive(ctx, func(msg inet256.Message) {
				n := appendHeaders(buf[:0], &msg)
				n += copy(buf[n:], msg.Payload)
				_, err := pconn.Write(buf[:n])
				if err != nil {
					sendErr = err
				}
			}); err != nil {
				return err
			}
			if sendErr != nil {
				return sendErr
			}
		}
	})
	return eg.Wait()
}

type MessageType uint32

const (
	MT_Data = MessageType(iota)

	MT_LookupPublicKey
	MT_PublicKey

	MT_FindAddr
	MT_Addr
)

const (
	MaxMsgLen = 4 + inet256.MaxMTU

	MinMsgLen     = 4
	MinMsgLenData = MinMsgLen + 32 + 32

	MinMsgLenFindAddr = 7
	MinMsgLenAddr     = MinMsgLen + 16 + 32

	MinMsgLenLookupPublicKey = 4 + 32
	MinMsgLenPublicKey       = 4 + 16
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
	case MT_FindAddr:
		if len(x) < MinMsgLenFindAddr {
			return nil, fmt.Errorf("too short to be FindAddr")
		}
	case MT_Addr:
		if len(x) < MinMsgLenAddr {
			return nil, fmt.Errorf("too short")
		}
	case MT_LookupPublicKey:
		if len(x) < MinMsgLenLookupPublicKey {
			return nil, fmt.Errorf("too short")
		}
	case MT_PublicKey:
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

func (m Message) GetSrc() inet256.Addr {
	return inet256.AddrFromBytes(m[4:])
}

func (m Message) GetDst() inet256.Addr {
	return inet256.AddrFromBytes(m[4+32:])
}

func (m Message) GetPayload() []byte {
	return m[4+32+32:]
}

func (m Message) GetRequestID() (ret [16]byte) {
	return *(*[16]byte)(m[4 : 4+16])
}

func (m Message) GetPublicKey() (inet256.PublicKey, error) {
	return inet256.ParsePublicKey(m[4+32:])
}

func (m Message) GetFindAddr() ([]byte, int) {
	numBits := binary.BigEndian.Uint16(m[4:6])
	prefix := m[6:]
	return prefix, int(numBits)
}

func (m Message) GetAddr() inet256.Addr {
	return inet256.AddrFromBytes(m[4:])
}
