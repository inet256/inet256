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
	fr := NewFrame()
	for i := 0; i <= MaxFrameBodyLen/100; i += 100 {
		fr.SetLen(i)
		if err := w.WriteFrame(ctx, fr); err != nil {
			require.NoError(t, err)
		}
	}
	require.Greater(t, buf.Len(), 0)
	// Read
	fr = NewFrame()
	for i := 0; i <= MaxFrameBodyLen/100; i += 100 {
		if err := r.ReadFrame(ctx, fr); err != nil {
			require.NoError(t, err)
		}
		require.Equal(t, i, fr.Len())
	}
	require.Equal(t, 0, buf.Len())
}
