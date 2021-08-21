package inet256

import (
	"errors"
	"fmt"
	"net"

	"github.com/brendoncarroll/go-p2p"
)

var (
	ErrPublicKeyNotFound = p2p.ErrPublicKeyNotFound
	ErrNoAddrWithPrefix  = errors.New("no address with prefix")
	ErrWouldBlock        = errors.New("call to Recv would block")
	ErrClosed            = net.ErrClosed
)

func IsPublicKeyNotFound(err error) bool {
	return err == ErrPublicKeyNotFound
}

func IsUnreachable(err error) bool {
	target := &ErrAddrUnreachable{}
	return errors.Is(err, target)
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

func IsErrWouldBlock(err error) bool {
	return err == ErrWouldBlock
}
