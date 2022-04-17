package bitstr

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuffer(t *testing.T) {
	x := make([]byte, 16)
	_, err := rand.Read(x[:])
	require.NoError(t, err)

	buf := Buffer{}
	buf.AppendAll(BytesMSB{Bytes: x})
	s := buf.BitString()
	buf2, _ := s.AsBytesMSB()
	require.Equal(t, x, buf2)
}
