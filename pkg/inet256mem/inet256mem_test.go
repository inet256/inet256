package inet256mem

import (
	"testing"

	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/inet256test"
)

func TestService(t *testing.T) {
	inet256test.TestService(t, func(t testing.TB, xs []inet256.Service) {
		x := New()
		for i := range xs {
			xs[i] = x
		}
	})
}

func BenchmarkService(b *testing.B) {
	inet256test.BenchService(b, func(t testing.TB, xs []inet256.Service) {
		x := New()
		for i := range xs {
			xs[i] = x
		}
	})
}
