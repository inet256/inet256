package forrestnet

import (
	"time"

	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/networks/nettmpl1"
	"github.com/inet256/inet256/pkg/inet256"
)

var _ networks.Network = &Network{}

type Addr = inet256.Addr

type Network struct {
	*nettmpl1.Network
	router *Router
}

func New(params networks.Params, roots []inet256.ID) *Network {
	r := NewRouter(params.Logger)
	nwk := nettmpl1.New(params, r, time.Second)
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
