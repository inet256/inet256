package inet256

import (
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/swarmtest"
)

func TestPeerSwarm(t *testing.T) {
	swarmtest.TestSuiteSecureSwarm(t, func(t testing.TB, n int) []p2p.SecureSwarm {
		r := memswarm.NewRealm()
		swarms := make([]p2p.SecureSwarm, n)
		for i := 0; i < n; i++ {
			pk := p2ptest.NewTestKey(t, i)
			swarms[i] = newPeerSwarm(r.NewSwarmWithKey(pk), NewPeerStore())
		}
		return swarms
	})
}
