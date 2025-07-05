package inet256mem

import (
	"testing"

	"go.inet256.org/inet256/src/inet256"
	"go.inet256.org/inet256/src/inet256tests"
)

func TestService(t *testing.T) {
	inet256tests.TestService(t, func(t testing.TB, xs []inet256.Service) {
		x := New()
		for i := range xs {
			xs[i] = x
		}
	})
}

func BenchmarkService(b *testing.B) {
	inet256tests.BenchService(b, func(t testing.TB, xs []inet256.Service) {
		x := New()
		for i := range xs {
			xs[i] = x
		}
	})
}
