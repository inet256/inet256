package inet256client

import (
	"net"
	"testing"

	"github.com/brendoncarroll/go-p2p/p2ptest"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/inet256test"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestDial(t *testing.T) {
	privateKey := p2ptest.NewTestKey(t, 0)
	serv := inet256srv.NewTestServer(t, inet256srv.OneHopFactory)
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
