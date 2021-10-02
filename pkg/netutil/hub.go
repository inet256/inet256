package netutil

import (
	"context"
	"sync"

	"github.com/inet256/inet256/pkg/inet256"
)

type Message = inet256.Message

type TellHub struct {
	recvs chan recvReq

	closeOnce sync.Once
	closed    chan struct{}
	err       error
}

func NewTellHub() *TellHub {
	return &TellHub{
		recvs:  make(chan recvReq),
		closed: make(chan struct{}),
	}
}

func (q *TellHub) Receive(ctx context.Context, fn func(inet256.Message)) error {
	if err := q.checkClosed(); err != nil {
		return err
	}
	req := recvReq{
		fn:   fn,
		done: make(chan struct{}),
	}
	select {
	case <-q.closed:
		return q.err
	case q.recvs <- req:
		// non-blocking case
	default:
		// blocking case
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-q.closed:
			return q.err
		case q.recvs <- req:
		}
	}
	// once we get to here we are committed
	<-req.done
	return nil
}

// Deliver delivers a message to a caller of Recv
// If Deliver returns an error it will be from the context expiring, or because the hub closed.
func (q *TellHub) Deliver(ctx context.Context, m Message) error {
	select {
	case <-q.closed:
		return q.err
	case <-ctx.Done():
		return ctx.Err()
	case req := <-q.recvs:
		// once we are here we are committed; no using the context
		defer close(req.done)
		req.fn(m)
		return nil
	}
}

func (q *TellHub) checkClosed() error {
	select {
	case <-q.closed:
		return q.err
	default:
		return nil
	}
}

func (q *TellHub) CloseWithError(err error) {
	if err == nil {
		err = inet256.ErrClosed
	}
	q.closeOnce.Do(func() {
		q.err = err
		close(q.closed)
	})
}

type recvReq struct {
	fn   func(inet256.Message)
	done chan struct{}
}
