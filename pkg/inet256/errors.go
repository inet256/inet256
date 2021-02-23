package inet256

import (
	"errors"
	"fmt"

	"github.com/brendoncarroll/go-p2p"
)

var (
	ErrPublicKeyNotFound = p2p.ErrPublicKeyNotFound
	ErrNoAddrWithPrefix  = errors.New("no address with prefix")
)

func IsPublicKeyNotFound(err error) bool {
	return err == ErrPublicKeyNotFound
}

func IsUnreachable(err error) bool {
	target := &ErrAddrUnreachable{}
	return errors.Is(err, target)
}

type ErrAddrUnreachable struct {
	Addr Addr
}

func (e ErrAddrUnreachable) Error() string {
	return fmt.Sprintf("address is unreachable: %v", e.Addr)
}
