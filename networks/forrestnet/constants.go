package forrestnet

import (
	"time"

	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/pkg/inet256"
)

const (
	LocationClaimTTL         = 60 * time.Second
	LocaitonClaimRenewPeriod = LocationClaimTTL / 3
)

var DefaultRoots = []inet256.Addr{
	allZeros(),
	allOnes(),
}

func Factory(params networks.Params) networks.Network {
	return New(params, DefaultRoots)
}
