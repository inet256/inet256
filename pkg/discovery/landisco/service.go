package landisco

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
)

type service struct {
	lookingFor map[p2p.PeerID][]string
}

func New() p2p.DiscoveryService {
	s := &service{
		lookingFor: make(map[p2p.PeerID][]string),
	}
	return s
}

func (s *service) Find(ctx context.Context, id p2p.PeerID) ([]string, error) {
	// TODO: implement
	return nil, nil
}

func (s *service) Announce(ctx context.Context, id p2p.PeerID, xs []string, ttl time.Duration) error {
	// TODO: implement
	return nil
}

func (s *service) Close() error {
	return nil
}
