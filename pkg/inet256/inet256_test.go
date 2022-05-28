package inet256

import (
	"crypto/ed25519"
	"crypto/rsa"
	"io"
	mrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddrMarshalText(t *testing.T) {
	rng := mrand.New(mrand.NewSource(0))
	const N = 10
	for i := 0; i < N; i++ {
		x := Addr{}
		_, err := rng.Read(x[:])
		require.NoError(t, err)
		data, err := x.MarshalText()
		// t.Logf(string(data))
		require.NoError(t, err)
		y := Addr{}
		require.NoError(t, y.UnmarshalText(data))
		require.Equal(t, x, y)
	}
}

func TestPublicKeyMarshal(t *testing.T) {
	t.Run("Ed25519", func(t *testing.T) {
		testPublicKeyMarshal(t, func(r io.Reader) PublicKey {
			pub, _, err := ed25519.GenerateKey(r)
			require.NoError(t, err)
			return pub
		})
	})
	t.Run("RSA", func(t *testing.T) {
		testPublicKeyMarshal(t, func(r io.Reader) PublicKey {
			priv, err := rsa.GenerateKey(r, 512)
			require.NoError(t, err)
			return priv.Public()
		})
	})
}

func testPublicKeyMarshal(t *testing.T, genKey func(io.Reader) PublicKey) {
	const N = 10
	for i := 0; i < N; i++ {
		rng := mrand.New(mrand.NewSource(int64(i)))
		x := genKey(rng)
		data := MarshalPublicKey(x)
		// t.Log(hex.EncodeToString(data))
		y, err := ParsePublicKey(data)
		require.NoError(t, err)
		require.Equal(t, x, y)
	}
}
