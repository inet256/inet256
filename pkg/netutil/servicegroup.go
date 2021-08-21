package netutil

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type ServiceGroup struct {
	setupOnce sync.Once
	ctx       context.Context
	cf        context.CancelFunc

	eg errgroup.Group
}

// Go runs fn in another go routine.
// When the ServiceGroup is stopped the context passed to fn will be cancelled.
// If fn ever returns an error other than ctx.Err(), it will be logged and the service will be restarted.
func (sg *ServiceGroup) Go(fn func(context.Context) error) {
	sg.setupOnce.Do(func() {
		sg.ctx, sg.cf = context.WithCancel(context.Background())
	})
	sg.eg.Go(func() error {
		for {
			err := fn(sg.ctx)
			if errors.Is(err, sg.ctx.Err()) {
				return nil
			}
			logrus.Errorf("service crashed with %v. restarting...", err)
			time.Sleep(time.Second)
		}
	})
}

func (sg *ServiceGroup) Stop() error {
	sg.cf()
	return sg.eg.Wait()
}
