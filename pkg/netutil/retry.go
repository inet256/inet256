package netutil

import (
	"context"
	"time"
)

type retryConfig struct {
	predicate  func(error) bool
	pulseTrain PulseTrain
}

type RetryOption = func(rc *retryConfig)

func WithPredicate(p func(error) bool) func(rc *retryConfig) {
	return func(rc *retryConfig) {
		rc.predicate = p
	}
}

func WithPulseTrain(pt PulseTrain) func(rc *retryConfig) {
	return func(rc *retryConfig) {
		rc.pulseTrain = pt
	}
}

// Retry calls fn until it returns nil.
// - To only retry on certain errors use WithPredicate to define a predicate.  True means retry.
// - To set the time between retries use WithPulseTrain and specify a pulse train.
func Retry(ctx context.Context, fn func() error, opts ...RetryOption) error {
	rc := retryConfig{
		predicate:  func(error) bool { return true },
		pulseTrain: NewLinear(time.Second),
	}
	for _, opt := range opts {
		opt(&rc)
	}

	for {
		if err := fn(); err == nil || !rc.predicate(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-rc.pulseTrain.Out():
		}
	}
}

// PulseTrain is a series of pulses over time.
// callers wait for the next pulse using <-Out()
type PulseTrain interface {
	Reset()
	Stop()
	Out() <-chan time.Time
}

// NewLinear creates a PulseTrain with pulses evenly spaced
func NewLinear(period time.Duration) PulseTrain {
	return linearPulseTrain{
		period: period,
		ticker: time.NewTicker(period),
	}
}

type linearPulseTrain struct {
	period time.Duration
	ticker *time.Ticker
}

func (lpt linearPulseTrain) Stop() {
	lpt.ticker.Stop()
}

func (lpt linearPulseTrain) Reset() {
	lpt.ticker.Reset(lpt.period)
}

func (lpt linearPulseTrain) Out() <-chan time.Time {
	return lpt.ticker.C
}
