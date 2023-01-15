package multihoming

import (
	"context"
	"crypto/ed25519"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/f/x509"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/swarmtest"
	"github.com/brendoncarroll/stdctx/logctx"
	"golang.org/x/exp/slog"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/peers"
)

func TestSwarm(t *testing.T) {
	//bgCtx := logctx.NewContext(context.Background(), slog.Default())
	swarmtest.TestSecureSwarm(t, func(t testing.TB, swarms []p2p.SecureSwarm[Addr, inet256.PublicKey]) {
		// r := memswarm.NewRealm[x509.PublicKey]()
		// for i := range swarms {
		// 	pk := newTestKey(t, i)
		// 	swarms[i] = New[memswarm.Addr](bgCtx, r.NewSwarmWithKey(pk), peers.ChainStore[memswarm.Addr]{})
		// }
	})
}

func newTestKey(t testing.TB, i int) x509.PublicKey {
	pk := p2ptest.NewTestKey(t, i)
	return x509.PublicKey{
		Algorithm: x509.Algo_Ed25519,
		Data:      []byte(pk.Public().(ed25519.PublicKey)),
	}
}
