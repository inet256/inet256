package beaconnet

import (
	"time"

	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/mesh256/routers"
)

const (
	defaultBeaconPeriod = 1 * time.Second
	defaultPeerStateTTL = 30 * time.Second
)

type Network = routers.Network

func Factory(params mesh256.NetworkParams) mesh256.Network {
	return NewNetwork(params)
}

func NewNetwork(params mesh256.NetworkParams) *Network {
	r := NewRouter()
	return routers.NewNetwork(params, r, time.Second)
}
