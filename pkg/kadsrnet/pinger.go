package kadsrnet

import (
	"context"
	"crypto/rand"
	sync "sync"
	"time"
)

type sendPingFunc = func(ctx context.Context, dst Addr, path Path, ping *Ping) error

type pingKey struct {
	dst  Addr
	uuid [16]byte
}

type pinger struct {
	send   sendPingFunc
	active sync.Map
}

func newPinger(spf sendPingFunc) *pinger {
	return &pinger{send: spf}
}

func (p *pinger) ping(ctx context.Context, dst Addr, path Path) (time.Duration, error) {
	startTime := time.Now()
	key := pingKey{
		dst:  dst,
		uuid: newUUID(),
	}
	ch := make(chan *Pong, 1)
	p.put(key, ch)
	defer p.delete(key)
	ping := &Ping{
		Timestamp: startTime.Unix(),
		Uuid:      key.uuid[:],
	}
	if err := p.send(ctx, dst, path, ping); err != nil {
		return -1, err
	}
	select {
	case <-ctx.Done():
		return -1, ctx.Err()
	case <-ch:
		endTime := time.Now()
		return endTime.Sub(startTime), nil
	}
}

func (p *pinger) handlePong(from Addr, pong *Pong) {
	if pong.Ping == nil {
		return
	}
	uuid := [16]byte{}
	copy(uuid[:], pong.Ping.Uuid)
	key := pingKey{
		dst:  from,
		uuid: uuid,
	}
	ch := p.get(key)
	ch <- pong
	close(ch)
}

func (p *pinger) put(key pingKey, ch chan *Pong) {
	p.active.Store(key, ch)
}

func (p *pinger) get(key pingKey) chan *Pong {
	v, ok := p.active.Load(key)
	if !ok {
		return nil
	}
	return v.(chan *Pong)
}

func (p *pinger) delete(key pingKey) {
	p.active.Delete(key)
}

func newUUID() [16]byte {
	uuid := [16]byte{}
	if n, err := rand.Read(uuid[:]); err != nil || n < len(uuid) {
		panic(err)
	}
	return uuid
}
