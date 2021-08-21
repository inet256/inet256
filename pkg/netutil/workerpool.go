package netutil

import (
	"context"
	"sync"
)

type WorkerPool struct {
	Fn func(ctx context.Context)

	mu      sync.Mutex
	workers []worker
}

func (wp *WorkerPool) SetCount(count int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	for len(wp.workers) != count {
		if len(wp.workers) < count {
			wp.spawn()
		} else {
			wp.despawn()
		}
	}
}

func (wp *WorkerPool) spawn() {
	ctx, cf := context.WithCancel(context.Background())
	w := worker{
		ctx:  ctx,
		cf:   cf,
		done: make(chan struct{}),
	}
	wp.workers = append(wp.workers, w)
	go w.run(wp.Fn)
}

func (wp *WorkerPool) despawn() {
	l := len(wp.workers)
	w := wp.workers[l-1]
	wp.workers = wp.workers[:l-1]
	w.cf()
	<-w.done
}

type worker struct {
	ctx  context.Context
	cf   context.CancelFunc
	done chan struct{}
}

func (w worker) run(fn func(context.Context)) {
	defer close(w.done)
	fn(w.ctx)
}
