package kadsrnet

import (
	"context"
	sync "sync"

	"github.com/pkg/errors"
)

type future struct {
	done   chan struct{}
	once   sync.Once
	result interface{}
	err    error
}

func newFuture() *future {
	return &future{
		done: make(chan struct{}),
	}
}

func (f *future) wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-f.done:
		return nil
	}
}

func (f *future) get(ctx context.Context) (interface{}, error) {
	if err := f.wait(ctx); err != nil {
		return nil, err
	}
	return f.result, f.err
}

func (f *future) complete(res interface{}) {
	f.once.Do(func() {
		f.result = res
		close(f.done)
	})
}

func (f *future) cancel() {
	f.once.Do(func() {
		f.err = errors.Errorf("future cancelled")
		close(f.done)
	})
}
