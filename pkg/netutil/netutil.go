package netutil

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p/s/swarmutil"
	"github.com/inet256/inet256/pkg/inet256"
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

type PulseTrain = swarmutil.PulseTrain

// NewLinear creates a PulseTrain with pulses evenly spaced
func NewLinear(period time.Duration) PulseTrain {
	return swarmutil.NewLinear(period)
}

type ErrList = swarmutil.ErrList

type TellHub = swarmutil.TellHub[inet256.Addr]

func NewTellHub() *TellHub {
	return swarmutil.NewTellHub[inet256.Addr]()
}