package netutil

import (
	"go.brendoncarroll.net/p2p/s/swarmutil"
	"go.inet256.org/inet256/src/inet256"
)

type ErrList = swarmutil.ErrList

type TellHub = swarmutil.TellHub[inet256.Addr]

func NewTellHub() TellHub {
	return swarmutil.NewTellHub[inet256.Addr]()
}

type Queue = swarmutil.Queue[inet256.Addr]

func NewQueue(maxLen int) Queue {
	return swarmutil.NewQueue[inet256.Addr](maxLen, inet256.MTU)
}
