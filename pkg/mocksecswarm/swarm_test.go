package mocksecswarm

import (
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
)

func TestSwarm(t *testing.T) {
	swarmtest.TestSwarm(t, func(t testing.TB, n int) []p2p.Swarm {
		r := memswarm.NewRealm()
		xs := make([]p2p.Swarm, n)
		for i := range xs {
			inner := r.NewSwarm()
			k := p2ptest.NewTestKey(t, i)
			xs[i] = New(inner, k)
		}
		t.Cleanup(func() { p2ptest.CloseSwarms(xs) })
		return xs
	})
}
