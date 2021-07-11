package inet256p2p

import (
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/swarmtest"
)

func TestAsker(t *testing.T) {
	t.Run("Swarm", func(t *testing.T) {
		swarmtest.TestSwarm(t, func(t testing.TB, xs []p2p.Swarm) {
			r := memswarm.NewRealm()
			for i := range xs {
				s := r.NewSwarm()
				xs[i] = newAsker(s)
			}
		})
	})
	t.Run("AskSwarm", func(t *testing.T) {
		swarmtest.TestAskSwarm(t, func(t testing.TB, xs []p2p.AskSwarm) {
			r := memswarm.NewRealm()
			for i := range xs {
				s := r.NewSwarm()
				xs[i] = newAsker(s)
			}
		})
	})
}
