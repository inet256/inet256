package inet256lb

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestUnixStream(t *testing.T) {
	p := filepath.Join(t.TempDir(), "test.socket")
	testStreamEndpoints(t, func(t testing.TB) (fe, be StreamEndpoint) {
		fe, err := NewUNIXStreamFrontend(p)
		require.NoError(t, err)
		t.Cleanup(func() { fe.Close() })
		be = NewUNIXStreamBackend(p)
		t.Cleanup(func() { be.Close() })
		return fe, be
	})
}

func testStreamEndpoints(t *testing.T, newEndpoints func(testing.TB) (fe, be StreamEndpoint)) {
	fe, be := newEndpoints(t)
	eg := errgroup.Group{}
	eg.Go(func() error {
		for {
			c, err := fe.Open(ctx)
			if err != nil {
				return err
			}
			t.Log("opened frontend connection from", c.RemoteAddr())
			defer c.Close()
			data, err := io.ReadAll(c)
			if err != nil {
				return err
			}
			t.Log("server read: " + string(data))
		}
	})
	eg.Go(func() error {
		c, err := be.Open(ctx)
		if err != nil {
			return err
		}
		t.Log("opened connection to backend to", c.RemoteAddr())
		defer c.Close()
		if _, err := c.Write([]byte("hello world")); err != nil {
			return err
		}
		t.Log("done writing data to backend")
		return c.Close()
	})
	require.NoError(t, eg.Wait())
}
