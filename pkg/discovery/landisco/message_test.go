package landisco

import (
	mrand "math/rand"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/stretchr/testify/require"
)

func TestMessage(t *testing.T) {
	ids := generatePeerIDs(10)
	ptext := "this is a test"
	m := NewMessage(ids[9], []byte(ptext))

	n, actualPtext, err := UnpackMessage(m, ids)
	require.NoError(t, err)
	require.Equal(t, ptext, string(actualPtext))
	require.Equal(t, 9, n)
}

func generatePeerIDs(n int) []p2p.PeerID {
	ids := make([]p2p.PeerID, n)
	for i := range ids {
		mrand.Read(ids[i][:])
	}
	return ids
}
