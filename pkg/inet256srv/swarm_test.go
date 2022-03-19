package inet256srv

import (
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/swarmtest"
)

func TestSwarm(t *testing.T) {
	swarmtest.TestSecureSwarm(t, func(t testing.TB, swarms []p2p.SecureSwarm[Addr]) {
		r := memswarm.NewRealm()
		for i := range swarms {
			pk := p2ptest.NewTestKey(t, i)
			swarms[i] = newSwarm[memswarm.Addr](r.NewSwarmWithKey(pk), ChainPeerStore[memswarm.Addr]{})
		}
	})
}
