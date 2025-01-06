package mesh256

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"go.brendoncarroll.net/p2p"
	"go.brendoncarroll.net/p2p/f/x509"
	"go.brendoncarroll.net/p2p/s/memswarm"
	"go.brendoncarroll.net/p2p/s/multiswarm"
	"go.brendoncarroll.net/stdctx/logctx"
	"go.uber.org/zap"

	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/inet256test"
	"go.inet256.org/inet256/pkg/mesh256/multihoming"
	"go.inet256.org/inet256/pkg/peers"
)

func NewTestServer(t testing.TB, nf NetworkFactory) *Server {
	l, _ := zap.NewDevelopment()
	ctx := logctx.NewContext(context.Background(), l)
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
	r := memswarm.NewSecureRealm[x509.PublicKey](memswarm.WithQueueLen(4))
	stores := make([]peers.Store[TransportAddr], len(xs))
	srvs := make([]*Server, len(xs))
	for i := range srvs {
		priv := inet256test.NewPrivateKey(t, math.MaxInt32+i)
		pubX509 := PublicKeyFromINET256(priv.Public())
		stores[i] = peers.NewStore[TransportAddr]()
		srvs[i] = NewServer(Params{
			Background: ctx,
			Swarms: map[string]multiswarm.DynSwarm{
				"external": multiswarm.WrapSecureSwarm[memswarm.Addr, x509.PublicKey](r.NewSwarm(pubX509)),
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
			peers.SetAddrs[multiswarm.Addr](stores[i], getMainAddr(srvs[j]), getTransportAddrs(srvs[j]))
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

func NewTestSwarm[A p2p.Addr](t testing.TB, x p2p.SecureSwarm[A, x509.PublicKey], peers peers.Store[A]) Swarm {
	sw := multihoming.New(multihoming.Params[A, x509.PublicKey, inet256.Addr]{
		Background: context.Background(),
		Inner:      x,
		Peers:      peers,
		GroupBy: func(pub x509.PublicKey) (inet256.Addr, error) {
			pub2, err := PublicKeyFromX509(pub)
			if err != nil {
				return inet256.Addr{}, err
			}
			return inet256.NewAddr(pub2), nil
		},
	})
	return swarm{sw}
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
