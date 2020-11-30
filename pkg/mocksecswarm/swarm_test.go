package mocksecswarm

import (
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/swarmtest"
	"github.com/stretchr/testify/require"
)

func TestSwarm(t *testing.T) {
	swarmtest.TestSuiteSwarm(t, func(t testing.TB, n int) []p2p.Swarm {
		r := memswarm.NewRealm()
		xs := make([]p2p.Swarm, n)
		for i := range xs {
			inner := r.NewSwarm()
			k := p2ptest.NewTestKey(t, i)
			xs[i] = New(inner, k)
		}
		t.Cleanup(func() {
			for i := range xs {
				require.Nil(t, xs[i].Close())
			}
		})
		return xs
	})
}
