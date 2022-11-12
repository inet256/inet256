package inet256ipc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamFramer(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewStreamFramer(buf)
	r := NewStreamFramer(buf)

	// Write
	fr := NewFrame()
	for i := 0; i <= MaxFrameBodyLen; i++ {
		fr.SetLen(i)
		if err := w.WriteFrame(ctx, fr); err != nil {
			require.NoError(t, err)
		}
	}
	require.Greater(t, buf.Len(), 0)
	// Read
	fr = NewFrame()
	for i := 0; i <= MaxFrameBodyLen; i++ {
		if err := r.ReadFrame(ctx, fr); err != nil {
			require.NoError(t, err)
		}
		require.Equal(t, i, fr.Len())
	}
	require.Equal(t, 0, buf.Len())
}
