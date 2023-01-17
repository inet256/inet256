package netutil

import (
	"context"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	ctx := context.Background()
	const N = 3
	q := NewQueue[inet256.Addr](N)

	for i := 0; i < N; i++ {
		require.True(t, q.Deliver(p2p.Message[inet256.Addr]{
			Payload: []byte("hello world"),
		}))
		require.Equal(t, i+1, q.Len())
	}
	require.False(t, q.Deliver(p2p.Message[inet256.Addr]{
		Payload: []byte("hello world"),
	}))

	var msg p2p.Message[inet256.Addr]
	for i := 0; i < N; i++ {
		require.Equal(t, N-i, q.Len())
		err := q.Read(ctx, &msg)
		require.NoError(t, err)
		require.Equal(t, "hello world", string(msg.Payload))
		require.Equal(t, N-i-1, q.Len())
	}
}
