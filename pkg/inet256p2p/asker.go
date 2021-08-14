package inet256p2p

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/swarmutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type asker struct {
	p2p.Swarm

	mu        sync.RWMutex
	counts    map[string]uint32
	responses map[responseKey]chan []byte

	askHub  *swarmutil.AskHub
	tellHub *swarmutil.TellHub
}

func newAsker(s p2p.Swarm) *asker {
	a := &asker{
		Swarm: s,

		counts:    make(map[string]uint32),
		responses: make(map[responseKey]chan []byte),
		askHub:    swarmutil.NewAskHub(),
		tellHub:   swarmutil.NewTellHub(),
	}
	go a.recvLoop(context.Background())
	return a
}

func (a *asker) recvLoop(ctx context.Context) error {
	buf := make([]byte, a.Swarm.MaxIncomingSize())
	for {
		var src, dst p2p.Addr
		n, err := a.Swarm.Receive(ctx, &src, &dst, buf)
		if err != nil {
			return err
		}
		data := buf[:n]
		if err := a.handleMessage(ctx, p2p.Message{
			Src:     src,
			Dst:     dst,
			Payload: data,
		}); err != nil {
			logrus.Error(err)
		}
	}
}

func (a *asker) Receive(ctx context.Context, src, dst *p2p.Addr, buf []byte) (int, error) {
	return a.tellHub.Receive(ctx, src, dst, buf)
}

func (a *asker) ServeAsk(ctx context.Context, fn p2p.AskHandler) error {
	return a.askHub.ServeAsk(ctx, fn)
}

func (a *asker) Tell(ctx context.Context, dst p2p.Addr, data p2p.IOVec) error {
	m := make(message, 8)
	m.setAsk(false)
	v := append(p2p.IOVec{m}, data...)
	return a.Swarm.Tell(ctx, dst, v)
}

func (a *asker) MTU(ctx context.Context, dst p2p.Addr) int {
	return a.Swarm.MTU(ctx, dst) - 8
}

func (a *asker) Ask(ctx context.Context, resp []byte, dst p2p.Addr, data p2p.IOVec) (int, error) {
	m := make(message, 8)
	m.setAsk(true)
	m.setResponse(false)
	vec := append(p2p.IOVec{m}, data...)

	a.mu.Lock()
	n := a.counts[dst.String()]
	a.counts[dst.String()]++
	m.setIndex(n)
	rkey := responseKey{
		Addr: dst.String(),
		N:    n,
	}
	ch := make(chan []byte, 1)
	a.responses[rkey] = ch
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		delete(a.responses, rkey)
	}()
	if err := a.Swarm.Tell(ctx, dst, vec); err != nil {
		return 0, err
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case data := <-ch:
		return copy(resp, data), nil
	}
}

func (a *asker) handleMessage(ctx context.Context, msg p2p.Message) error {
	m := message(msg.Payload)
	if !m.isValidLen() {
		return errors.Errorf("got too short message from %v", msg.Src)
	}
	src := msg.Src

	switch {
	// tell
	case !m.isAsk():
		msg.Payload = m.getPayload()
		return a.tellHub.Deliver(ctx, msg)

	// request
	case m.isAsk() && !m.isResponse():
		msg2 := msg
		msg2.Payload = m.getPayload()
		resp := newMessage()
		resp.setAsk(true)
		resp.setResponse(true)
		resp.setIndex(m.index())
		respBuf := make([]byte, a.Swarm.MaxIncomingSize()-8)
		n, err := a.askHub.Deliver(ctx, respBuf, msg2)
		if err != nil {
			return err
		}
		resp = append(resp, respBuf[:n]...)
		return a.Swarm.Tell(ctx, src, p2p.IOVec{resp})

	// response
	case m.isAsk():
		m = append([]byte{}, m...) // copy m
		key := responseKey{
			Addr: src.String(),
			N:    m.index(),
		}
		a.mu.Lock()
		mb, exists := a.responses[key]
		delete(a.responses, key)
		if exists {
			mb <- m.getPayload()
			close(mb)
		}
		a.mu.Unlock()
	}
	return nil
}

type responseKey struct {
	Addr string
	N    uint32
}

// message format
// 0			4			8			N
//  ..		    | 	index	| payload 	|
// 1st bit is set to 0 for tells, 1 for asks
// 2nd bit is set to 0 for requests, 1 for responses. It is meaningless if 1st bit is not set.
// last 4 bytes are index
type message []byte

func newMessage() message {
	return make(message, 8)
}

func (m message) isValidLen() bool {
	return len(m) >= 8
}

func (m message) isAsk() bool {
	return m[0]&0x01 > 0
}

func (m message) setAsk(x bool) {
	if x {
		m[0] |= 0x01
	} else {
		m[0] &= (0x01 ^ 0xff)
	}
}

func (m message) isResponse() bool {
	return (m[0] & 0x02) > 0
}

func (m message) setResponse(x bool) {
	if x {
		m[0] |= 0x02
	} else {
		m[0] &= (0x02 ^ 0xff)
	}
}

func (m message) index() uint32 {
	return binary.BigEndian.Uint32(m[4:8])
}

func (m message) setIndex(x uint32) {
	binary.BigEndian.PutUint32(m[4:8], x)
}

func (m message) getPayload() []byte {
	return m[8:]
}

func (m message) String() string {
	isAsk := m.isAsk()
	return fmt.Sprintf("message{isAsk: %v, size: %d}", isAsk, len(m))
}
