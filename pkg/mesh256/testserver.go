package mesh256

import (
	"math"
	"testing"

	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/inet256/inet256/pkg/peers"
	"github.com/stretchr/testify/require"
)

func NewTestServer(t testing.TB, nf NetworkFactory) *Server {
	pk := inet256test.NewPrivateKey(t, math.MaxInt32)
	ps := peers.NewStore[TransportAddr]()
	s := NewServer(Params{
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
	r := memswarm.NewRealm()
	stores := make([]peers.Store[TransportAddr], len(xs))
	srvs := make([]*Server, len(xs))
	for i := range srvs {
		pk := inet256test.NewPrivateKey(t, math.MaxInt32+i)
		stores[i] = peers.NewStore[TransportAddr]()
		srvs[i] = NewServer(Params{
			Swarms: map[string]multiswarm.DynSwarm{
				"external": multiswarm.WrapSecureSwarm[memswarm.Addr](r.NewSwarmWithKey(pk.BuiltIn())),
			},
			NewNetwork: nf,
			Peers:      stores[i],
			PrivateKey: pk,
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
	addr, err := x.MainAddr()
	if err != nil {
		panic(err)
	}
	return addr
}

func getTransportAddrs(x *Server) []TransportAddr {
	addrs, err := x.TransportAddrs()
	if err != nil {
		panic(err)
	}
	return addrs
}
