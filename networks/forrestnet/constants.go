package forrestnet

import (
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
)

const (
	LocationClaimTTL         = 60 * time.Second
	LocaitonClaimRenewPeriod = LocationClaimTTL / 3
)

var DefaultRoots = []inet256.Addr{
	allZeros(),
	allOnes(),
}

func Factory(params mesh256.NetworkParams) mesh256.Network {
	return New(params, DefaultRoots)
}
