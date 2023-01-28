package inet256ipc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamFramer(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewStreamFramer(nil, buf)
	r := NewStreamFramer(buf, nil)

	// Write
	sendbuf := [MaxMessageLen]byte{}
	for i := 0; i <= MaxMessageLen/100; i += 100 {
		if err := w.Send(ctx, sendbuf[:i*100]); err != nil {
			require.NoError(t, err)
		}
	}
	require.Greater(t, buf.Len(), 0)
	// Read
	for i := 0; i <= MaxMessageLen/100; i += 100 {
		if err := r.Receive(ctx, func(data []byte) {
			require.Equal(t, i*100, len(data))
		}); err != nil {
			require.NoError(t, err)
		}
	}
	require.Equal(t, 0, buf.Len())
}
