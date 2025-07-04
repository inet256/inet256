package inet256

import (
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand"
	"testing"

	dilithium2 "github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/cloudflare/circl/sign/ed25519"
	"github.com/stretchr/testify/require"
	inet256crypto "go.inet256.org/inet256/src/internal/crypto"
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
		newDilithium2(t, 0),
		inet256crypto.ProductSigner{newEd25519(t, 0), newDilithium2(t, 0)},
	}
	for _, privKey := range privKeys {
		privKey := privKey
		t.Run(fmt.Sprintf("%T", privKey), func(t *testing.T) {
			t.Parallel()
			data := []byte("test data")
			purpose := SigCtxString("test purpose")
			pki := DefaultPKI
			sig := pki.Sign(&purpose, privKey, data, nil)
			ok := pki.Verify(&purpose, privKey.Public().(PublicKey), data, sig)
			require.True(t, ok)
		})
	}
}

func newEd25519(t testing.TB, i int) PrivateKey {
	rng := mrand.New(mrand.NewSource(int64(i)))
	_, priv, err := ed25519.GenerateKey(rng)
	require.NoError(t, err)
	return priv
}

func newDilithium2(t testing.TB, i int) PrivateKey {
	seed := make([]byte, 32)
	binary.LittleEndian.PutUint64(seed[:8], uint64(i))
	_, privKey := dilithium2.Scheme().DeriveKey(seed)
	return privKey
}
