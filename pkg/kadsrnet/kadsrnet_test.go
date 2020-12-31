package kadsrnet

import (
	"context"
	"testing"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
)

func TestNetwork(t *testing.T) {
	inet256.TestSuite(t, func(params inet256.NetworkParams) inet256.Network {
		n := New(params.PrivateKey, params.Swarm, params.Peers)
		err := n.WaitStartup(context.TODO())
		require.NoError(t, err)
		return n
	})
}
