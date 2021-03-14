package inet256

import (
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

const (
	stateWaiting = 0
	stateRunning = 1
)

type RecvHub struct {
	state uint32
	ready chan struct{}
	fn    RecvFunc

	closeOnce sync.Once
	done      chan struct{}
	err       error
}

func NewRecvHub() *RecvHub {
	return &RecvHub{
		ready: make(chan struct{}),
		done:  make(chan struct{}),
	}
}

func (h *RecvHub) Recv(fn RecvFunc) error {
	if !atomic.CompareAndSwapUint32(&h.state, stateWaiting, stateRunning) {
		return errors.Errorf("already receiving")
	}
	h.fn = fn
	close(h.ready)
	<-h.done
	return h.err
}

func (h *RecvHub) Deliver(src, dst Addr, data []byte) {
	select {
	case <-h.done:
		return
	default:
		select {
		case <-h.done:
			return
		case <-h.ready:
			h.fn(src, dst, data)
		}
	}
}

func (h *RecvHub) CloseWithError(err error) {
	h.closeOnce.Do(func() {
		h.err = err
		close(h.done)
	})
}
