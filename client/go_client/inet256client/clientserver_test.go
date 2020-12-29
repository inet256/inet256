package inet256client

import (
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
	n := inet256.NewNode(inet256.Params{
		PrivateKey: privateKey,
		Networks: []inet256.NetworkSpec{
			{
				Name:    "",
				Factory: inet256.OneHopFactory,
			},
		},
		Swarms: map[string]p2p.SecureSwarm{
			"virtual": mr.NewSwarmWithKey(privateKey),
		},
	})
	s := inet256grpc.NewServer(n)
	gs := grpc.NewServer()
	inet256grpc.RegisterINET256Server(gs, s)
	l, err := net.Listen("tcp", "127.0.0.1:25600")
	require.NoError(t, err)
	eg := errgroup.Group{}
	eg.Go(func() error {
		return gs.Serve(l)
	})
	eg.Go(func() error {
		defer l.Close()
		defer gs.Stop()
		// time.Sleep(30 * time.Millisecond)
		privateKey := p2ptest.NewTestKey(t, 1)
		c, err := New("127.0.0.1:25600", privateKey)
		if err != nil {
			return err
		}
		defer c.Close()
		t.Log(c)
		return nil
	})
	require.NoError(t, eg.Wait())
}
