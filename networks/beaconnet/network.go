package beaconnet

import (
	"time"

	"github.com/inet256/inet256/networks/neteng"
	"github.com/inet256/inet256/pkg/mesh256"
)

const (
	defaultBeaconPeriod = 1 * time.Second
	defaultPeerStateTTL = 30 * time.Second
)

type Network struct {
	*neteng.Network
	router neteng.Router
}

func Factory(params mesh256.NetworkParams) mesh256.Network {
	return New(params)
}

func New(params mesh256.NetworkParams) *Network {
	router := NewRouter(params.Logger)
	nw := neteng.New(params, router, time.Second)
	return &Network{
		Network: nw,
		router:  router,
	}
}
