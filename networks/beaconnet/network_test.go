package beaconnet

import (
	"testing"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/mesh256/mesh256test"
)

func TestNetwork(t *testing.T) {
	mesh256test.TestNetwork(t, Factory)
}

func TestServer(t *testing.T) {
	inet256test.TestService(t, func(t testing.TB, xs []inet256.Service) {
		mesh256.NewTestServers(t, Factory, xs)
	})
}

func BenchmarkService(b *testing.B) {
	inet256test.BenchService(b, func(t testing.TB, xs []inet256.Service) {
		mesh256.NewTestServers(t, Factory, xs)
	})
}
