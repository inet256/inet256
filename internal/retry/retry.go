package retry

import (
	"context"
	"time"

	"go.brendoncarroll.net/p2p/s/swarmutil/retry"
)

type RetryOption = retry.RetryOption

func Retry(ctx context.Context, fn func() error, opts ...RetryOption) error {
	return retry.Retry(ctx, fn, opts...)
}

func RetryRet1[T any](ctx context.Context, fn func() (T, error), opts ...RetryOption) (T, error) {
	return retry.RetryRet1(ctx, fn, opts...)
}

func WithBackoff(bf retry.BackoffFunc) retry.RetryOption {
	return retry.WithBackoff(bf)
}

func NewConstantBackoff(d time.Duration) retry.BackoffFunc {
	return retry.NewConstantBackoff(d)
}

func WithPredicate(fn func(error) bool) retry.RetryOption {
	return retry.WithPredicate(fn)
}
