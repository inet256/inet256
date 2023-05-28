package peers

import (
	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

type Addr = inet256.Addr

type Info[T p2p.Addr] struct {
	Addrs []T
}

type Getter[T p2p.Addr] interface {
	Get(x Addr) (Info[T], bool)
}

type Updater[T p2p.Addr] interface {
	Update(x Addr, fn func(Info[T]) Info[T])
}

// PeerStore stores information about peers
// TA is the type of the transport address
type Store[T p2p.Addr] interface {
	Set

	Add(x Addr)
	Remove(x Addr)
	Getter[T]
	Updater[T]
}

func ListAddrs[T p2p.Addr](s Getter[T], k Addr) []T {
	info, ok := s.Get(k)
	if !ok {
		return nil
	}
	return info.Addrs
}

func SetAddrs[T p2p.Addr](s Updater[T], k Addr, addrs []T) {
	s.Update(k, func(x Info[T]) Info[T] {
		x.Addrs = append([]T{}, addrs...)
		return x
	})
}

// PeerSet represents a set of peers
type Set interface {
	List() []Addr
	Contains(Addr) bool
}
