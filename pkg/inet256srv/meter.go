package inet256srv

import (
	"sync"
	"sync/atomic"
)

type meter struct {
	tx, rx sync.Map
}

func (m *meter) Tx(k Addr, x int) uint64 {
	actual, _ := m.tx.LoadOrStore(k, new(uint64))
	return atomic.AddUint64(actual.(*uint64), uint64(x))
}

func (m *meter) Rx(k Addr, x int) uint64 {
	actual, _ := m.rx.LoadOrStore(k, new(uint64))
	return atomic.AddUint64(actual.(*uint64), uint64(x))
}
