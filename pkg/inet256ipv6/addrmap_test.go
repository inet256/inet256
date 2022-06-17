package inet256ipv6

import (
	"math/rand"
	"testing"

	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/inet256/inet256/pkg/bitstr"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompress(t *testing.T) {
	const N = 20
	for i := 0; i < N; i++ {
		buf := [32]byte{}
		_, err := rand.Read(buf[:])
		require.NoError(t, err)
		for j := 0; j < N; j++ {
			buf[j/8] &= ^(0x80 >> (j % 8))
		}

		x := bitstr.FromSource(bitstr.BytesMSB{buf[:], 0, 256})
		t.Logf("INPUT: %s", x.String())
		scratch := bitstr.Buffer{}
		compress(&scratch, x)
		c := scratch.BitString()
		t.Logf("COMPRESS: %s", c.String())
		y := uncompress(c)
		t.Logf("OUTPUT: %s", y.String())
		require.Equal(t, x, y)
	}
}

func Test256To6(t *testing.T) {
	const N = 10000
	for i := 0; i < N; i++ {
		addr := inet256.Addr{}
		_, err := rand.Read(addr[:])
		require.NoError(t, err)
		lz := kademlia.LeadingZeros(addr[:])

		ip := IPv6FromINET256(addr)
		require.True(t, netPrefix.Contains(ip))
		prefix, nbits, err := INET256PrefixFromIPv6(ip)
		require.NoError(t, err)
		require.True(t, inet256.HasPrefix(addr[:], prefix, nbits), "ADDR: %x PREFIX: %x", addr[:], prefix)
		require.Equal(t, nbits, lz+(128-NetworkPrefix().Bits()-7)+1)
	}
}

func TestUint7(t *testing.T) {
	for i := uint8(0); i < 128; i++ {
		x := i
		buf := bitstr.Buffer{}
		writeUint7(&buf, uint8(x))
		require.Equal(t, buf.Len(), 7)
		y := readUint7(buf.BitString())
		assert.Equal(t, x, y)
	}
}
