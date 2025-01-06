package netutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.brendoncarroll.net/p2p"
	"go.inet256.org/inet256/pkg/inet256"
)

func TestQueue(t *testing.T) {
	ctx := context.Background()
	const N = 3
	q := NewQueue(N)
	defer q.Close()

	for i := 0; i < N; i++ {
		require.Equal(t, i, q.Len())
		require.True(t, q.Deliver(p2p.Message[inet256.Addr]{
			Payload: []byte("hello world"),
		}))
		require.Equal(t, i+1, q.Len())
	}
	for i := 0; i < 2; i++ {
		require.False(t, q.Deliver(p2p.Message[inet256.Addr]{
			Payload: []byte("goodbye world"),
		}))
	}
	for i := 0; i < N; i++ {
		require.Equal(t, N-i, q.Len())
		err := q.Receive(ctx, func(m p2p.Message[inet256.Addr]) {
			require.Equal(t, "hello world", string(m.Payload))
		})
		require.NoError(t, err)
		require.Equal(t, N-i-1, q.Len())
	}
}

func BenchmarkQueue(b *testing.B) {
	b.Run("SendRecv", func(b *testing.B) {
		ctx := context.Background()
		const msgSize = 1 << 15
		q := NewQueue(1)

		msgIn := p2p.Message[inet256.Addr]{
			Payload: make([]byte, msgSize),
		}
		b.ResetTimer()
		b.ReportAllocs()
		b.SetBytes(msgSize)
		var count int
		for i := 0; i < b.N; i++ {
			if accepted := q.Deliver(msgIn); !accepted {
				require.True(b, accepted)
			}
			if err := q.Receive(ctx, func(msg p2p.Message[inet256.Addr]) {
				// just something so this isn't optimized away
				count += int(msg.Payload[0]) + int(msg.Payload[len(msg.Payload)-1])
			}); err != nil {
				require.NoError(b, err)
			}
		}
	})
}
