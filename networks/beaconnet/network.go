package beaconnet

import (
	"time"

	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/networks/neteng"
)

const (
	defaultBeaconPeriod = 1 * time.Second
	defaultPeerStateTTL = 30 * time.Second
)

type Network struct {
	*neteng.Network
	router neteng.Router
}

func Factory(params networks.Params) networks.Network {
	return New(params)
}

func New(params networks.Params) *Network {
	router := NewRouter(params.Logger)
	nw := neteng.New(params, router, time.Second)
	return &Network{
		Network: nw,
		router:  router,
	}
}
