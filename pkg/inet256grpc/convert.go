package inet256grpc

import (
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
)

func peerStatusFromProto(xs []*PeerStatus) []inet256srv.PeerStatus {
	ys := make([]inet256srv.PeerStatus, len(xs))
	for i := range xs {
		lastSeen := make(map[string]time.Time, len(xs[i].LastSeen))
		for k, v := range xs[i].LastSeen {
			lastSeen[k] = time.Unix(v, 0)
		}
		ys[i] = inet256srv.PeerStatus{
			Addr:     inet256.AddrFromBytes(xs[i].Addr),
			LastSeen: lastSeen,
		}
	}
	return ys
}

func peerStatusToProto(xs []inet256srv.PeerStatus) []*PeerStatus {
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
