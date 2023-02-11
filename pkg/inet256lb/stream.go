package inet256lb

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/brendoncarroll/stdctx/logctx"
	"golang.org/x/exp/slog"
)

type StreamEndpoint interface {
	Open(ctx context.Context) (net.Conn, error)
	Close() error
}

type StreamBalancer struct {
	mu       sync.RWMutex
	backends map[string]*streamBalEntry
}

func NewStreamBalancer() *StreamBalancer {
	return &StreamBalancer{}
}

func (b *StreamBalancer) AddBackend(k string, be StreamEndpoint) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.backends == nil {
		b.backends = make(map[string]*streamBalEntry)
	}
	if _, exists := b.backends[k]; exists {
		return errors.New("backend already exists")
	}
	b.backends[k] = &streamBalEntry{backend: be}
	return nil
}

func (b *StreamBalancer) RemoveBackend(k string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.backends[k]; !exists {
		return errors.New("backend does not exist")
	}
	delete(b.backends, k)
	return nil
}

func (b *StreamBalancer) ServeFrontend(ctx context.Context, frontend StreamEndpoint) error {
	wg := sync.WaitGroup{}
	defer wg.Wait()
	for {
		fconn, err := frontend.Open(ctx)
		if err != nil {
			return err
		}
		logctx.Info(ctx, "accepted connection", slog.Any("remote_addr", fconn.RemoteAddr()))

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer logctx.Info(ctx, "closed connection", slog.Any("remote_addr", fconn.RemoteAddr()))
			if err := b.serveFrontendConn(ctx, fconn); err != nil {
				logctx.Errorln(ctx, err)
			}
		}()
	}
}

func (b *StreamBalancer) serveFrontendConn(ctx context.Context, fconn net.Conn) error {
	ent, err := b.pickBackend(fconn.RemoteAddr(), fconn.LocalAddr())
	if err != nil {
		return err
	}
	ent.active.Add(1)
	defer ent.active.Add(-1)
	bconn, err := ent.backend.Open(ctx)
	if err != nil {
		return fmt.Errorf("could not connect to backend %w", err)
	}
	return b.serveStream(ctx, fconn, bconn)
}

func (b *StreamBalancer) GetActiveCounts() map[string]int64 {
	ret := make(map[string]int64)
	b.mu.RLock()
	defer b.mu.RUnlock()
	for k, be := range b.backends {
		ret[k] = be.active.Load()
	}
	return ret
}

func (b *StreamBalancer) pickBackend(raddr, laddr net.Addr) (*streamBalEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.backends) == 0 {
		return nil, errors.New("no backends available")
	}
	var best *streamBalEntry
	var minActive int64
	for _, e := range b.backends {
		if active := e.active.Load(); best == nil || active < minActive {
			best = e
			minActive = active
		}
	}
	return best, nil
}

func (b *StreamBalancer) serveStream(ctx context.Context, fconn, bconn net.Conn) error {
	return PlumbRWC(fconn, bconn)
}

type streamBalEntry struct {
	active  atomic.Int64
	backend StreamEndpoint
}
