package kadsrnet

import (
	"testing"

	"github.com/inet256/inet256/pkg/inet256"
)

func TestNetwork(t *testing.T) {
	inet256.TestSuite(t, func(params inet256.NetworkParams) inet256.Network {
		n := New(params.PrivateKey, params.Swarm, params.Peers, params.Logger)
		return n
	})
}
