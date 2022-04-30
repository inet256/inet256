// package futures provides the Future type which can be used
// to model the result of a ongoing computation which could fail.
//
// Futures are not the idiomatic way to deal with concurrency in Go.
// Go APIs should be synchronous not asynchronous.
// The network *is* asynchronous, futures provide a way to build synchronous APIs
// on top of the asynchronous network
package futures

import (
	"context"
	"sync"
)

// Future represents a value of type T which will be delivered by some process which can fail.
type Future[T any] struct {
	x    T
	err  error
	once sync.Once
	done chan struct{}
}

func New[T any]() *Future[T] {
	return &Future[T]{
		done: make(chan struct{}, 1),
	}
}

func (f *Future[T]) Succeed(x T) (ret bool) {
	f.once.Do(func() {
		ret = true
		f.x = x
		close(f.done)
	})
	return ret
}

func (f *Future[T]) Fail(err error) (ret bool) {
	f.once.Do(func() {
		ret = true
		f.err = err
		close(f.done)
	})
	return ret
}

// Await blocks until the future has completed.
// If the future fails, Await returns an error and the zero value of T.
// If the future succeeds, the error will be nil.
// If the context expires Await returns ctx.Err()
func (f *Future[T]) Await(ctx context.Context) (ret T, _ error) {
	select {
	case <-ctx.Done():
		return ret, ctx.Err()
	case <-f.done:
		return f.x, f.err
	}
}

func (f *Future[T]) IsDone() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}
