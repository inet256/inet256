package forrestnet

import (
	"time"

	"github.com/inet256/inet256/networks/neteng"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
)

var _ mesh256.Network = &Network{}

type Addr = inet256.Addr

type Network struct {
	*neteng.Network
	router *Router
}

func New(params mesh256.NetworkParams, roots []inet256.ID) *Network {
	r := NewRouter(params.Logger)
	nwk := neteng.New(params, r, time.Second)
	return &Network{
		Network: nwk,
		router:  r,
	}
}

func allZeros() inet256.Addr {
	return inet256.Addr{}
}

func allOnes() inet256.Addr {
	a := inet256.Addr{}
	for i := range a {
		a[i] = 0xff
	}
	return a
}

func getLeafAddr(b *Beacon) inet256.Addr {
	return inet256.NewAddrPKIX(b.Chain[len(b.Chain)-1].PublicKey)
}
