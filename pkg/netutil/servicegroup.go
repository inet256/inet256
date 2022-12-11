package netutil

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/brendoncarroll/stdctx/logctx"
	"golang.org/x/sync/errgroup"
)

type ServiceGroup struct {
	Background context.Context

	setupOnce sync.Once
	ctx       context.Context
	cf        context.CancelFunc

	eg errgroup.Group
}

// Go runs fn in another go routine.
// When the ServiceGroup is stopped the context passed to fn will be cancelled.
// If fn ever returns an error other than ctx.Err(), it will be logged.
// The service will be restarted, unless the group has been stopped.
func (sg *ServiceGroup) Go(fn func(context.Context) error) {
	bgCtx := sg.Background
	if bgCtx == nil {
		bgCtx = context.Background()
	}
	sg.setupOnce.Do(func() {
		sg.ctx, sg.cf = context.WithCancel(bgCtx)
	})
	sg.eg.Go(func() error {
		ctx := sg.ctx
		for {
			err := fn(sg.ctx)
			if errors.Is(err, sg.ctx.Err()) {
				return nil
			}
			if isContextDone(sg.ctx) {
				logctx.Errorf(ctx, "while stopping service group: %v", err)
				return nil
			}
			logctx.Errorf(ctx, "service crashed with %v. restarting...", err)
			time.Sleep(time.Second)
		}
	})
}

func (sg *ServiceGroup) Stop() error {
	sg.cf()
	return sg.eg.Wait()
}

func isContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
