package inet256

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHub(t *testing.T) {
	hub := NewTellHub()
	ctx := context.Background()
	go func() {
		for i := 0; i < 100; i++ {
			hub.Deliver(ctx, Message{})
		}
	}()

	require.NoError(t, hub.Wait(ctx))
	var src, dst Addr
	buf := make([]byte, 100)
	_, err := hub.Recv(ctx, &src, &dst, buf)
	require.NoError(t, err)
}
