package inet256d

import (
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
)

type Server struct {
	UnimplementedAdminServer
}

func peerStatusFromProto(xs []*PeerStatus) []mesh256.PeerStatus {
	ys := make([]mesh256.PeerStatus, len(xs))
	for i := range xs {
		lastSeen := make(map[string]time.Time, len(xs[i].LastSeen))
		for k, v := range xs[i].LastSeen {
			lastSeen[k] = time.Unix(v, 0)
		}
		ys[i] = mesh256.PeerStatus{
			Addr:     inet256.AddrFromBytes(xs[i].Addr),
			LastSeen: lastSeen,
		}
	}
	return ys
}

func peerStatusToProto(xs []mesh256.PeerStatus) []*PeerStatus {
	ys := make([]*PeerStatus, len(xs))
	for i := range xs {
		lastSeen := make(map[string]int64, len(xs[i].LastSeen))
		for k, v := range xs[i].LastSeen {
			lastSeen[k] = v.Unix()
		}
		ys[i] = &PeerStatus{
			Addr:     xs[i].Addr[:],
			LastSeen: lastSeen,
		}
	}
	return ys
}
