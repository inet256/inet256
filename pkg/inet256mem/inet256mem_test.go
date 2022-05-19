package inet256mem

import (
	"testing"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
)

func TestService(t *testing.T) {
	inet256test.TestService(t, func(t testing.TB, xs []inet256.Service) {
		x := New()
		for i := range xs {
			xs[i] = x
		}
	})
}
