package mesh256

import (
	"sync"
	"sync/atomic"
)

type meterSet struct {
	m sync.Map
}

func (m meterSet) Tx(k Addr, x int) uint64 {
	actual, _ := m.m.LoadOrStore(k, new(meter))
	return actual.(*meter).Tx(x)
}

func (m meterSet) Rx(k Addr, x int) uint64 {
	actual, _ := m.m.LoadOrStore(k, new(meter))
	return actual.(*meter).Rx(x)
}

type meter struct {
	rx, tx uint64
}

func (m *meter) Tx(x int) uint64 {
	return atomic.AddUint64(&m.tx, uint64(x))
}

func (m *meter) Rx(x int) uint64 {
	return atomic.AddUint64(&m.rx, uint64(x))
}
