package inet256http

import (
	"context"
	"net/http"
	"testing"

	"github.com/inet256/inet256/networks/beaconnet"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/inet256test"
)

func TestClientServer(t *testing.T) {
	inet256test.TestService(t, func(t *testing.T, xs []inet256.Service) {
		inet256srv.NewTestServers(t, beaconnet.Factory, xs)
		for i := range xs {
			srv := NewServer(xs[i])
			hs := http.Server{Addr: "127.0.0.1:", Handler: srv}
			go func() {
				hs.ListenAndServe()
			}()
			t.Cleanup(func() { hs.Shutdown(context.Background()) })
			client := NewClient(hs.Addr)
			xs[i] = client
		}
	})
}
