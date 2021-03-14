package inet256p2p

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/swarmutil"
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
	go s.ServeTells(a.fromBelow)
	return a
}

func (a *asker) ServeTells(fn p2p.TellHandler) error {
	return a.tellHub.ServeTells(fn)
}

func (a *asker) ServeAsks(fn p2p.AskHandler) error {
	return a.askHub.ServeAsks(fn)
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

func (a *asker) Ask(ctx context.Context, dst p2p.Addr, data p2p.IOVec) ([]byte, error) {
	m := make(message, 8)
	m.setAsk(true)
	m.setResponse(false)
	vec := append(p2p.IOVec{m}, data...)

	a.mu.Lock()
	n := a.counts[dst.Key()]
	a.counts[dst.Key()]++
	m.setIndex(n)
	rkey := responseKey{
		Addr: dst.Key(),
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
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data := <-ch:
		return data, nil
	}
}

func (a *asker) fromBelow(msg *p2p.Message) {
	m := message(msg.Payload)
	if !m.isValidLen() {
		logrus.Warn("got too short message from ", msg.Src)
	}
	src := msg.Src

	switch {
	// tell
	case !m.isAsk():
		msg.Payload = m.getPayload()
		a.tellHub.DeliverTell(msg)

	// request
	case m.isAsk() && !m.isResponse():
		// copy message
		msg2 := *msg
		m = append([]byte{}, m...) // copy m
		msg2.Payload = m.getPayload()
		go func() {
			ctx := context.Background()
			//	ctx, cf := context.WithTimeout(ctx, 10*time.Second)
			//defer cf()
			resp := newMessage()
			resp.setAsk(true)
			resp.setResponse(true)
			resp.setIndex(m.index())
			buf := bytes.Buffer{}
			buf.Write(resp[:8])
			a.askHub.DeliverAsk(ctx, &msg2, &buf)
			if err := a.Swarm.Tell(ctx, src, p2p.IOVec{buf.Bytes()}); err != nil {
				logrus.Error(err)
			}
		}()

	// response
	case m.isAsk():
		m = append([]byte{}, m...) // copy m
		key := responseKey{
			Addr: src.Key(),
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
}

type responseKey struct {
	Addr string
	N    uint32
}

type askResp struct {
	data []byte
	err  error
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
