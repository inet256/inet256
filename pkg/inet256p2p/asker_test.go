package inet256p2p

import (
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/swarmtest"
)

func TestAsker(t *testing.T) {
	t.Run("TestSuiteSwarm", func(t *testing.T) {
		swarmtest.TestSuiteSwarm(t, func(t testing.TB, n int) []p2p.Swarm {
			xs := make([]p2p.Swarm, n)
			r := memswarm.NewRealm()
			for i := range xs {
				s := r.NewSwarm()
				xs[i] = newAsker(s)
			}
			return xs
		})
	})
	t.Run("TestSuiteAskSwarm", func(t *testing.T) {
		swarmtest.TestSuiteAskSwarm(t, func(t testing.TB, n int) []p2p.AskSwarm {
			xs := make([]p2p.AskSwarm, n)
			r := memswarm.NewRealm()
			for i := range xs {
				s := r.NewSwarm()
				xs[i] = newAsker(s)
			}
			return xs
		})
	})
}
