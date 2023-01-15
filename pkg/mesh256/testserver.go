package mesh256

import (
	"context"
	"math"
	"testing"

	"github.com/brendoncarroll/go-p2p/f/x509"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/inet256/inet256/pkg/peers"
)

func NewTestServer(t testing.TB, nf NetworkFactory) *Server {
	ctx := logctx.NewContext(context.Background(), slog.Default())
	ctx, cf := context.WithCancel(ctx)
	t.Cleanup(cf)
	pk := inet256test.NewPrivateKey(t, math.MaxInt32)
	ps := peers.NewStore[TransportAddr]()
	s := NewServer(Params{
		Background: ctx,
		NewNetwork: nf,
		Peers:      ps,
		PrivateKey: pk,
	})
	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})
	return s
}

func NewTestServers(t testing.TB, nf NetworkFactory, xs []inet256.Service) {
	ctx := context.Background()
	ctx, cf := context.WithCancel(ctx)
	t.Cleanup(cf)
	r := memswarm.NewRealm[x509.PublicKey]()
	stores := make([]peers.Store[TransportAddr], len(xs))
	srvs := make([]*Server, len(xs))
	for i := range srvs {
		priv := inet256test.NewPrivateKey(t, math.MaxInt32+i)
		pubX509 := PublicKeyFromINET256(priv.Public())
		stores[i] = peers.NewStore[TransportAddr]()
		srvs[i] = NewServer(Params{
			Background: ctx,
			Swarms: map[string]multiswarm.DynSwarm{
				"external": multiswarm.WrapSecureSwarm[memswarm.Addr, x509.PublicKey](r.NewSwarmWithKey(pubX509)),
			},
			NewNetwork: nf,
			Peers:      stores[i],
			PrivateKey: priv,
		})
	}
	for i := range srvs {
		for j := range srvs {
			if i == j {
				continue
			}
			stores[i].Add(getMainAddr(srvs[j]))
			stores[i].SetAddrs(getMainAddr(srvs[j]), getTransportAddrs(srvs[j]))
		}
	}
	t.Cleanup(func() {
		for _, s := range srvs {
			require.NoError(t, s.Close())
		}
	})
	for i := range xs {
		xs[i] = srvs[i]
	}
}

func getMainAddr(x *Server) inet256.Addr {
	addr, err := x.MainAddr(context.TODO())
	if err != nil {
		panic(err)
	}
	return addr
}

func getTransportAddrs(x *Server) []TransportAddr {
	addrs, err := x.TransportAddrs(context.TODO())
	if err != nil {
		panic(err)
	}
	return addrs
}
