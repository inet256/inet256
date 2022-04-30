package netutil

import (
	"github.com/brendoncarroll/go-p2p/s/swarmutil"
	"github.com/inet256/inet256/pkg/inet256"
)

type ErrList = swarmutil.ErrList

type TellHub = swarmutil.TellHub[inet256.Addr]

func NewTellHub() *TellHub {
	return swarmutil.NewTellHub[inet256.Addr]()
}
