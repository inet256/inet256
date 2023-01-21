package netutil

import (
	"context"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

type Queue[A p2p.Addr] struct {
	queue     chan p2p.Message[A]
	freelist  chan p2p.Message[A]
	closeOnce sync.Once
	closed    chan struct{}
}

func NewQueue[A p2p.Addr](maxLen int) Queue[A] {
	if maxLen < 1 {
		panic(maxLen)
	}
	mtu := inet256.MaxMTU
	freelist := make(chan p2p.Message[A], maxLen)
	for i := 0; i < maxLen; i++ {
		freelist <- p2p.Message[A]{
			Payload: make([]byte, 0, mtu),
		}
	}
	return Queue[A]{
		queue:    make(chan p2p.Message[A], maxLen),
		freelist: freelist,
		closed:   make(chan struct{}),
	}
}

// Deliver does not block. It immediately returns true if there was room in the queue.
func (q *Queue[A]) Deliver(m p2p.Message[A]) bool {
	select {
	case <-q.closed:
		return false
	case m2 := <-q.freelist:
		copyMessage(&m2, &m)
		select {
		case q.queue <- m2:
			return true
		default:
			panic("queue is full, but freelist gave us a message")
		}
	default:
		return false
	}
}

func (q *Queue[A]) Receive(ctx context.Context, fn func(p2p.Message[A])) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-q.closed:
		return inet256.ErrClosed
	case msg := <-q.queue:
		fn(msg)
		zeroMessage(&msg)
		q.freelist <- msg
		return nil
	}
}

// Purge empties the queue and returns the number purged.
func (q *Queue[A]) Purge() (count int) {
	for len(q.queue) > 0 {
		m := <-q.queue
		zeroMessage[A](&m)
		q.freelist <- m
		count++
	}
	return count
}

func (q *Queue[A]) Close() error {
	q.closeOnce.Do(func() {
		// close so anyone in a select can bail out
		close(q.closed)
		// there should be cap(q.freelist) messages.  Let's get all of them and give them back to the void.
		for i := 0; i < cap(q.freelist); i++ {
			<-q.freelist
		}
		// After doing that there should be nothing in the queue, unless there are bootleg messages in circulation.
		if len(q.queue) != 0 {
			panic("there are still items in the queue after emptying freelist")
		}
	})
	return nil
}

func (q *Queue[A]) IsClosed() bool {
	select {
	case <-q.closed:
		return true
	default:
		return false
	}
}

func (q *Queue[A]) Len() int {
	return len(q.queue)
}

func (q *Queue[A]) IsFull() bool {
	return len(q.queue) == cap(q.queue)
}

func zeroMessage[A p2p.Addr](m *p2p.Message[A]) {
	var zero A
	m.Src = zero
	m.Dst = zero
	m.Payload = m.Payload[:0]
}

func copyMessage[A p2p.Addr](dst, src *p2p.Message[A]) {
	dst.Src = src.Src
	dst.Dst = src.Dst
	dst.Payload = append(dst.Payload[:0], src.Payload...)
}
