package inet256client

import (
	"context"
	"net"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

func TestDial(t *testing.T) {
	mr := memswarm.NewRealm()
	privateKey := p2ptest.NewTestKey(t, 0)
	serv := inet256.NewServer(inet256.Params{
		PrivateKey: privateKey,
		Networks: []inet256.NetworkSpec{
			{
				Name:    "",
				Factory: inet256.OneHopFactory,
			},
		},
		Peers: inet256.NewPeerStore(),
		Swarms: map[string]p2p.SecureSwarm{
			"virtual": mr.NewSwarmWithKey(privateKey),
		},
	})
	s := inet256grpc.NewServer(serv)
	gs := grpc.NewServer()
	inet256grpc.RegisterINET256Server(gs, s)
	l, err := net.Listen("tcp", "127.0.0.1:25600")
	require.NoError(t, err)
	ctx := context.Background()
	eg := errgroup.Group{}
	eg.Go(func() error {
		return gs.Serve(l)
	})
	eg.Go(func() error {
		defer l.Close()
		defer gs.Stop()
		privateKey := p2ptest.NewTestKey(t, 1)
		c, err := NewNode("127.0.0.1:25600", privateKey)
		if err != nil {
			return err
		}
		done := make(chan struct{})
		c.OnRecv(func(dst, src inet256.Addr, payload []byte) {
			done <- struct{}{}
		})
		err = c.Tell(ctx, p2p.NewPeerID(privateKey.Public()), []byte("this shouldn't break"))
		if err != nil {
			return err
		}
		select {
		case <-done:
		}
		defer c.Close()
		return nil
	})
	require.NoError(t, eg.Wait())
}
