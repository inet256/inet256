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
	ErrClosed            = net.ErrClosed
)

func IsErrPublicKeyNotFound(err error) bool {
	return err == ErrPublicKeyNotFound
}

func IsErrUnreachable(err error) bool {
	target := ErrAddrUnreachable{}
	return errors.As(err, &target)
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
