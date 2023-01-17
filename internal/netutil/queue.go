package netutil

import (
	"context"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"golang.org/x/exp/constraints"
)

type Queue[A p2p.Addr] struct {
	mu           sync.Mutex
	front, back  uint32
	nonEmpty     bool
	buffer       []p2p.Message[A]
	nonEmptyChan chan struct{}
}

func NewQueue[A p2p.Addr](maxLen int) Queue[A] {
	if maxLen < 1 {
		panic(maxLen)
	}
	return Queue[A]{
		buffer:       make([]p2p.Message[A], maxLen),
		front:        0,
		back:         0,
		nonEmptyChan: make(chan struct{}),
	}
}

// Deliver does not block. It immediately returns true if there was room in the queue.
func (q *Queue[A]) Deliver(m p2p.Message[A]) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.isFull() {
		return false
	}
	idx := q.pushBack()
	copyMessage(&q.buffer[idx], &m)
	return true
}

// Read reads a message from the front of the queue into msg, overwriting whatever was there.
// msg.Payload is truncated and reused to prevent allocating additional memory.
func (q *Queue[A]) Read(ctx context.Context, msg *p2p.Message[A]) error {
	for {
		if waitChan := func() chan struct{} {
			q.mu.Lock()
			defer q.mu.Unlock()
			if !q.isEmpty() {
				idx := q.popFront()
				copyMessage[A](msg, &q.buffer[idx])
				zeroMessage[A](&q.buffer[idx])
				if q.isEmpty() {
					close(q.nonEmptyChan)
					q.nonEmptyChan = make(chan struct{})
				}
				return nil
			} else {
				return q.nonEmptyChan
			}
		}(); waitChan == nil {
			return nil
		} else {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-waitChan:
			}
		}
	}
}

// Purge empties the queue and returns the number purged.
func (q *Queue[A]) Purge() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	count := q.len()
	q.front = 0
	q.back = 0
	q.nonEmpty = false
	close(q.nonEmptyChan)
	q.nonEmptyChan = make(chan struct{})
	return count
}

func (q *Queue[A]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.len()
}

func (q *Queue[A]) popFront() int {
	if q.isEmpty() {
		panic("pop on empty queue")
	}
	idx := q.front
	q.front = mod(q.front+1, uint32(len(q.buffer)))
	if mod(q.back-q.front, uint32(len(q.buffer))) == 0 {
		q.nonEmpty = false
	}
	return int(idx)
}

func (q *Queue[A]) pushBack() int {
	if q.isFull() {
		panic("push on full queue")
	}
	idx := q.back
	q.back = mod(q.back+1, uint32(len(q.buffer)))
	if mod(q.back-q.front, uint32(len(q.buffer))) == 0 {
		q.nonEmpty = true
	}
	return int(idx)
}

func (q *Queue[A]) isFull() bool {
	return mod(q.back-q.front, uint32(len(q.buffer))) == 0 && q.nonEmpty
}

func (q *Queue[A]) isEmpty() bool {
	return q.back == q.front && !q.nonEmpty
}

func (q *Queue[A]) len() int {
	l := mod(int(q.back)-int(q.front), len(q.buffer))
	if l == 0 && q.nonEmpty {
		l = len(q.buffer)
	}
	return l
}

func zeroMessage[A p2p.Addr](m *p2p.Message[A]) {
	var zero A
	m.Src = zero
	m.Dst = zero
	m.Payload = m.Payload[:0]
}

func mod[T constraints.Integer](x, m T) T {
	z := x % m
	if z < 0 {
		z += m
	}
	return z
}

func copyMessage[A p2p.Addr](dst, src *p2p.Message[A]) {
	dst.Src = src.Src
	dst.Dst = src.Dst
	dst.Payload = append(dst.Payload[:0], src.Payload...)
}
