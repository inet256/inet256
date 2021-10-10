package inet256client

import (
	"net"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestDial(t *testing.T) {
	mr := memswarm.NewRealm()
	privateKey := p2ptest.NewTestKey(t, 0)
	serv := inet256srv.NewServer(inet256srv.Params{
		PrivateKey: privateKey,
		Networks: map[inet256srv.NetworkCode]inet256.NetworkFactory{
			{}: inet256srv.OneHopFactory,
		},
		Peers: inet256srv.NewPeerStore(),
		Swarms: map[string]p2p.Swarm{
			"virtual": mr.NewSwarmWithKey(privateKey),
		},
	})
	s := inet256grpc.NewServer(serv)
	gs := grpc.NewServer()
	inet256grpc.RegisterINET256Server(gs, s)
	l, err := net.Listen("tcp", defaultAPIAddr)
	require.NoError(t, err)
	go gs.Serve(l)
	c, err := NewNode(defaultAPIAddr, privateKey)
	require.NoError(t, err)
	inet256test.TestSendRecvOne(t, c, c)
}
