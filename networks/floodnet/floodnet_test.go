package floodnet

import (
	"testing"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/inet256test"
)

func TestNetwork(t *testing.T) {
	inet256test.TestNetwork(t, Factory)
}

func TestServer(t *testing.T) {
	inet256test.TestService(t, func(t *testing.T, xs []inet256.Service) {
		inet256srv.NewTestServers(t, Factory, xs)
	})
}
