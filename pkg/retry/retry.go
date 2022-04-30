package retry

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p/s/swarmutil"
)

type RetryOption = swarmutil.RetryOption

func WithPredicate(p func(error) bool) RetryOption {
	return swarmutil.WithPredicate(p)
}

func WithPulseTrain(pt PulseTrain) RetryOption {
	return swarmutil.WithPulseTrain(pt)
}

func Retry(ctx context.Context, fn func() error, opts ...RetryOption) error {
	return swarmutil.Retry(ctx, fn, opts...)
}

func Retry1[T any](ctx context.Context, fn func() (T, error), opts ...RetryOption) (ret T, retErr error) {
	if err := Retry(ctx, func() error {
		var err error
		ret, err = fn()
		return err
	}); err != nil {
		var zero T
		return zero, err
	}
	return ret, nil
}

type PulseTrain = swarmutil.PulseTrain

// NewLinear creates a PulseTrain with pulses evenly spaced
func NewLinear(period time.Duration) PulseTrain {
	return swarmutil.NewLinear(period)
}
