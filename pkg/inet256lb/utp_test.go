package loadbal

import (
	"context"
	"io"
	"testing"

	"github.com/inet256/inet256/pkg/inet256mem"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

var ctx = context.Background()

func TestUTPConnect(t *testing.T) {
	s := inet256mem.New()
	n1 := inet256test.OpenNode(t, s, 0)
	n2 := inet256test.OpenNode(t, s, 1)

	eg := errgroup.Group{}
	eg.Go(func() error {
		l, err := ListenUTP(ctx, n1)
		if err != nil {
			return err
		}
		defer l.Close()
		t.Log("server listening")
		c, err := l.Accept()
		if err != nil {
			return err
		}
		defer c.Close()
		t.Log("accepted connection from", c.RemoteAddr())
		data, err := io.ReadAll(c)
		if err != nil {
			return err
		}
		t.Log("server received:", string(data))
		return nil
	})
	eg.Go(func() error {
		c, err := DialUTP(ctx, n2, n1.LocalAddr())
		if err != nil {
			return err
		}
		defer c.Close()
		t.Log("dialer connected")
		if _, err := c.Write([]byte("hello ")); err != nil {
			return err
		}
		if _, err := c.Write([]byte("world ")); err != nil {
			return err
		}
		return c.Close()
	})

	require.NoError(t, eg.Wait())
}
