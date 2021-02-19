package celldisco

import (
	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/d/celltracker"
)

func New(token string) (p2p.DiscoveryService, error) {
	return celltracker.NewClient(token)
}
