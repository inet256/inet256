package inet256

import (
	"errors"
	"fmt"
	"net"

	"go.brendoncarroll.net/p2p"
)

var (
	ErrPublicKeyNotFound = p2p.ErrPublicKeyNotFound
	ErrNoAddrWithPrefix  = errors.New("no address with prefix")
	ErrClosed            = net.ErrClosed
)

func IsErrPublicKeyNotFound(err error) bool {
	return err == ErrPublicKeyNotFound
}

func IsErrClosed(err error) bool {
	return errors.Is(err, ErrClosed)
}

type ErrAddrUnreachable struct {
	Addr Addr
}

func (e ErrAddrUnreachable) Error() string {
	return fmt.Sprintf("address is unreachable: %v", e.Addr)
}

func IsErrAddrUnreachable(err error) bool {
	return errors.As(err, &ErrAddrUnreachable{})
}

type ErrMTUExceeded struct {
	MessageLen int
}

func (e ErrMTUExceeded) Error() string {
	return fmt.Sprintf("message exceeds MTU. len=%d, mtu=%d", e.MessageLen, MTU)
}

func IsErrMTUExceeded(err error) bool {
	return errors.As(err, &ErrMTUExceeded{})
}
