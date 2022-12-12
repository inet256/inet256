package inet256

import (
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"
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
			pub2, err := PublicKeyFromBuiltIn(pub)
			require.NoError(t, err)
			return pub2
		})
	})
	t.Run("RSA", func(t *testing.T) {
		testPublicKeyMarshal(t, func(r io.Reader) PublicKey {
			priv, err := rsa.GenerateKey(r, 1024)
			require.NoError(t, err)
			pub, err := PublicKeyFromBuiltIn(priv.Public())
			require.NoError(t, err)
			return pub
		})
	})
}

func testPublicKeyMarshal(t *testing.T, genKey func(io.Reader) PublicKey) {
	const N = 10
	for i := 0; i < N; i++ {
		rng := mrand.New(mrand.NewSource(int64(i)))
		x := genKey(rng)
		data := MarshalPublicKey(nil, x)
		// t.Log(hex.EncodeToString(data))
		y, err := ParsePublicKey(data)
		require.NoError(t, err)
		require.Equal(t, x, y)
	}
}

func TestSignVerify(t *testing.T) {
	t.Parallel()
	privKeys := []PrivateKey{
		newEd25519(t, 0),
		newRSA(t, 0),
	}
	for _, privKey := range privKeys {
		t.Run(fmt.Sprintf("%T", privKey), func(t *testing.T) {
			t.Parallel()
			data := []byte("test data")
			purpose := "test purpose"
			sig := Sign(nil, privKey, purpose, data)
			ok := Verify(privKey.Public(), purpose, data, sig)
			require.True(t, ok)
		})
	}
}

func newEd25519(t testing.TB, i int) PrivateKey {
	rng := mrand.New(mrand.NewSource(int64(i)))
	_, priv, err := ed25519.GenerateKey(rng)
	require.NoError(t, err)
	priv2, err := PrivateKeyFromBuiltIn(priv)
	require.NoError(t, err)
	return priv2
}

func newRSA(t testing.TB, i int) PrivateKey {
	rng := mrand.New(mrand.NewSource(int64(i)))
	priv, err := rsa.GenerateKey(rng, 1024)
	require.NoError(t, err)
	priv2, err := PrivateKeyFromBuiltIn(priv)
	require.NoError(t, err)
	return priv2
}
