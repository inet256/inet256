package netutil

import (
	"context"
	"io"
	"sync"

	"github.com/inet256/inet256/pkg/inet256"
	"golang.org/x/sync/errgroup"
)

type Message = inet256.Message

type TellHub struct {
	recvs chan *recvReq
	ready readySwitch

	closeOnce sync.Once
	closed    chan struct{}
	err       error
}

func NewTellHub() *TellHub {
	closed := make(chan struct{})
	return &TellHub{
		ready:  newReadySwitch(closed),
		recvs:  make(chan *recvReq),
		closed: closed,
	}
}

func (q *TellHub) Receive(ctx context.Context, src, dst *inet256.Addr, buf []byte) (int, error) {
	if err := q.checkClosed(); err != nil {
		return 0, err
	}
	req := &recvReq{
		buf:  buf,
		src:  src,
		dst:  dst,
		done: make(chan struct{}, 1),
	}
	select {
	case <-q.closed:
		return 0, q.err
	case q.recvs <- req:
		// non-blocking case
	default:
		// blocking case
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-q.closed:
			return 0, q.err
		case q.recvs <- req:
		}
	}
	// once we get to here we are committed, unless the whole thing is closed we have to wait
	select {
	case <-q.closed:
		return 0, q.err
	case <-req.done:
		return req.n, req.err
	}
}

func (q *TellHub) Wait(ctx context.Context) error {
	if err := q.checkClosed(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-q.closed:
		return q.err
	case <-q.ready.Out():
		return nil
	}
}

// Deliver delivers a message to a caller of Recv
// If Deliver returns an error it will be from the context expiring, or because the hub closed.
func (q *TellHub) Deliver(ctx context.Context, m Message) error {
	return q.claim(ctx, func(src, dst *inet256.Addr, buf []byte) (int, error) {
		if len(buf) < len(m.Payload) {
			return 0, io.ErrShortBuffer
		}
		*src = m.Src
		*dst = m.Dst
		return copy(buf, m.Payload), nil
	})
}

// Claim calls fn, as if from a caller of Recv
// fn should never block
func (q *TellHub) claim(ctx context.Context, fn func(src, dst *inet256.Addr, buf []byte) (int, error)) error {
	// mark ready, until claim returns
	q.ready.MoreReady()
	defer q.ready.LessReady()
	// wait for a request
	select {
	case <-q.closed:
		return q.err
	case <-ctx.Done():
		return ctx.Err()
	case req := <-q.recvs:
		// once we are here we are committed no using the context
		// req.done is buffered and should never block anyway
		req.n, req.err = fn(req.src, req.dst, req.buf)
		close(req.done)
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
	q.closeOnce.Do(func() {
		q.err = err
		close(q.closed)
	})
}

type readySwitch struct {
	out        chan struct{}
	readyEdges chan bool
	closed     chan struct{}
}

func newReadySwitch(closed chan struct{}) readySwitch {
	rs := readySwitch{
		out:        make(chan struct{}),
		readyEdges: make(chan bool, 8),
		closed:     closed,
	}
	go rs.loop()
	return rs
}

func (rs readySwitch) loop() {
	var count int
	for {
		if count > 0 {
			select {
			case <-rs.closed:
				return
			case delta := <-rs.readyEdges:
				if delta {
					count++
				} else {
					count--
				}
			case rs.out <- struct{}{}:
			}
		} else {
			select {
			case <-rs.closed:
				return
			case delta := <-rs.readyEdges:
				if delta {
					count++
				} else {
					count--
				}
			}
		}
	}
}

func (rs readySwitch) Out() <-chan struct{} {
	return rs.out
}

func (rs readySwitch) MoreReady() {
	select {
	case rs.readyEdges <- true:
	case <-rs.closed:
	}
}

func (rs readySwitch) LessReady() {
	select {
	case rs.readyEdges <- false:
	case <-rs.closed:
	}
}

type recvReq struct {
	src, dst *inet256.Addr
	buf      []byte

	done chan struct{}
	n    int
	err  error
}

// Select waits on each network in xs and returns either:
// the index of a ready network, or -1 if the context expires.
func Select(ctx context.Context, xs []inet256.Network) int {
	s := NewSelector(xs)
	defer s.Close()
	return s.Which(ctx)
}

// Selector lets callers wait for 1 of N Networks to be ready to receive data
type Selector struct {
	ready    chan int
	networks []inet256.Network
	eg       *errgroup.Group
	cf       context.CancelFunc
}

func NewSelector(ns []inet256.Network) *Selector {
	ctx, cf := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)
	s := &Selector{
		ready:    make(chan int),
		networks: ns,
		cf:       cf,
		eg:       eg,
	}
	for i, n := range ns {
		i := i
		n := n
		eg.Go(func() error {
			for {
				err := n.WaitReceive(ctx)
				if err != nil {
					return err
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case s.ready <- i:
				}
			}
		})
	}
	return s
}

func (s *Selector) Receive(ctx context.Context, src, dst *inet256.Addr, buf []byte) (int, error) {
	var n int
	var err error
	for {
		x := s.Which(ctx)
		if x < 0 {
			return 0, ctx.Err()
		}
		n, err = inet256.ReceiveNonBlocking(s.networks[x], src, dst, buf)
		if !inet256.IsErrWouldBlock(err) {
			return n, err
		}
	}
}

func (s *Selector) Which(ctx context.Context) int {
	select {
	case x := <-s.ready:
		return x
	default:
	}
	select {
	case <-ctx.Done():
		return -1
	case x := <-s.ready:
		return x
	}
}

func (s *Selector) Wait(ctx context.Context) error {
	x := s.Which(ctx)
	if x < 0 {
		return ctx.Err()
	}
	return nil
}

func (s *Selector) Close() error {
	s.cf()
	return nil
}
