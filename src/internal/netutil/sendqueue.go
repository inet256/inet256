package netutil

import (
	"context"
	"net"

	"go.brendoncarroll.net/p2p"
	"go.inet256.org/inet256/src/inet256"
)

type TellFunc = func(ctx context.Context, dst inet256.Addr, m p2p.IOVec) error

// SendQueue queues messages, and sends them in the background.
type SendQueue struct {
	todo chan inet256.Message
	send TellFunc
	sg   ServiceGroup
	done chan struct{}
}

func NewSendQueue(depth int, send TellFunc) *SendQueue {
	sq := &SendQueue{
		todo: make(chan inet256.Message, depth),
		send: send,
		done: make(chan struct{}),
	}
	sq.sg.Go(sq.run)
	sq.sg.Go(func(ctx context.Context) error {
		defer close(sq.done)
		<-ctx.Done()
		return ctx.Err()
	})
	return sq
}

func (sq *SendQueue) run(ctx context.Context) error {
	for {
		select {
		case msg := <-sq.todo:
			if err := sq.send(ctx, msg.Dst, p2p.IOVec{msg.Payload}); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (sq *SendQueue) Tell(ctx context.Context, dst inet256.Addr, m p2p.IOVec) error {
	msg := inet256.Message{
		Dst:     dst,
		Payload: p2p.VecBytes(nil, m),
	}
	select {
	case <-sq.done:
		return net.ErrClosed
	case sq.todo <- msg:
		return nil
	}
}

func (sq *SendQueue) Close() error {
	return sq.sg.Stop()
}
