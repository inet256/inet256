package landisco

import (
	mrand "math/rand"
	"testing"

	"github.com/brendoncarroll/go-tai64"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
)

func TestMessage(t *testing.T) {
	ids := generatePeerIDs(10)
	ptext := "this is a test"
	m := NewMessage(tai64.Now(), ids[9], []byte(ptext))

	n, actualPtext, err := m.Open(ids)
	require.NoError(t, err)
	require.Equal(t, ptext, string(actualPtext))
	require.Equal(t, 9, n)
}

func generatePeerIDs(n int) []inet256.Addr {
	ids := make([]inet256.Addr, n)
	for i := range ids {
		mrand.Read(ids[i][:])
	}
	return ids
}
