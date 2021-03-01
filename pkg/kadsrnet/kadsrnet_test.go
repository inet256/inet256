package kadsrnet

import (
	"testing"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
)

func TestNetwork(t *testing.T) {
	inet256test.TestNetwork(t, func(params inet256.NetworkParams) inet256.Network {
		n := New(params.PrivateKey, params.Swarm, params.Peers, params.Logger)
		return n
	})
}

func TestServer(t *testing.T) {
	inet256test.TestServer(t, func(params inet256.NetworkParams) inet256.Network {
		n := New(params.PrivateKey, params.Swarm, params.Peers, params.Logger)
		return n
	})
}
