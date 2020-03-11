package inet256ipv6

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/require"
)

func TestCompress(t *testing.T) {
	const N = 20
	for i := 0; i < N; i++ {
		addr := inet256.Addr{}
		_, err := rand.Read(addr[:])
		require.Nil(t, err)
		for j := 0; j < N; j++ {
			addr[j/8] &= ^(0x80 >> (j % 8))
		}

		c := compress(addr[:])
		t.Logf("ADDR: %x", addr[:])
		t.Log("LZs:", leading0s(addr[:]))
		t.Logf("COMPRESS: %x", c)
		prefix, n := uncompress(c)
		prefix = prefix[:n/8-1]
		require.True(t, bytes.HasPrefix(addr[:], prefix), "ADDR: %x PREFIX: %x", addr[:], prefix)
	}
}

func TestBoth(t *testing.T) {
	const N = 10
	for i := 0; i < N; i++ {
		addr := inet256.Addr{}
		_, err := rand.Read(addr[:])
		require.Nil(t, err)

		ip := INet256ToIPv6(addr)
		t.Logf("IP: %v", ip)
		prefix, n := IPv6ToPrefix(ip)
		prefix = prefix[:n/8-1]
		require.True(t, bytes.HasPrefix(addr[:], prefix), "ADDR: %x PREFIX: %x", addr[:], prefix)
	}
}
