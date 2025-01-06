package inet256lb

import (
	"context"
	"testing"

	"go.inet256.org/inet256/pkg/inet256mem"
	"go.inet256.org/inet256/pkg/inet256test"
)

var ctx = context.Background()

func TestUTPEndpoint(t *testing.T) {
	testStreamEndpoints(t, func(t testing.TB) (fe, be StreamEndpoint) {
		s := inet256mem.New()
		n1 := inet256test.OpenNode(t, s, 0)
		n2 := inet256test.OpenNode(t, s, 1)
		fe = NewUTPFrontend(n1)
		be = NewUTPBackend(n2, n1.LocalAddr())
		return fe, be
	})
}
