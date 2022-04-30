package forrestnet

import (
	"github.com/inet256/inet256/pkg/inet256"
)

type ForrestState struct {
	trees map[inet256.Addr]*TreeState
}

func NewForrestState(privateKey inet256.PrivateKey, optimums ...inet256.Addr) *ForrestState {
	trees := map[inet256.Addr]*TreeState{}
	for _, addr := range optimums {
		trees[addr] = NewTreeState(privateKey, addr)
	}
	return &ForrestState{
		trees: trees,
	}
}

// Deliver is called with a Beacon, and a list of peers potentiall downstream from the beacon
// Deliver returns a slice of new beacons to be sent out.
// This will either be:
// - an ammended version of b sent to all downstream peers
// - an ammended version of a beacon already received from elsewhere sent just to `from`.
func (s *ForrestState) Deliver(from Peer, b *Beacon, downstream []Peer) (beacons []*Beacon, _ error) {
	root := inet256.AddrFromBytes(b.Optimum)
	tstate, exists := s.trees[root]
	if !exists {
		return nil, nil
	}
	return tstate.Deliver(from, b, downstream)
}

// Heartbeat is called periodically to contruct new beacons when no messages have been received.
func (s *ForrestState) Heartbeat(now Timestamp, peers []Peer) (bcasts []*Beacon) {
	for _, tstate := range s.trees {
		bcasts = append(bcasts, tstate.Heartbeat(now, peers)...)
	}
	return bcasts
}

func (s *ForrestState) LocationClaims() (ret []*LocationClaim) {
	for _, ts := range s.trees {
		if lc := ts.LocationClaim(); lc != nil {
			ret = append(ret, lc)
		}
	}
	return ret
}
