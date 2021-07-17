package discovery

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

type Service interface {
	Find(ctx context.Context, x inet256.Addr) ([]p2p.Addr, error)
	Announce(ctx context.Context, x inet256.Addr, addrs []p2p.Addr, ttl time.Duration) error
}
