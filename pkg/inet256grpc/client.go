package inet256grpc

import (
	context "context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func NewExtendedClient(endpoint string) (mesh256.Service, error) {
	gc, err := dial(endpoint)
	if err != nil {
		return nil, err
	}
	return &client{
		inetClient:  NewINET256Client(gc),
		adminClient: NewAdminClient(gc),
		log:         logrus.StandardLogger(),
	}, nil
}

func NewClient(endpoint string) (inet256.Service, error) {
	return NewExtendedClient(endpoint)
}

func dial(endpoint string) (*grpc.ClientConn, error) {
	return grpc.Dial(endpoint, grpc.WithInsecure())
}

type client struct {
	inetClient  INET256Client
	adminClient AdminClient
	log         *logrus.Logger
}

func (c *client) Open(ctx context.Context, privKey p2p.PrivateKey, opts ...inet256.NodeOption) (inet256.Node, error) {
	n, err := newNode(c.inetClient, privKey)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (c *client) Delete(ctx context.Context, privKey p2p.PrivateKey) error {
	_, err := c.inetClient.Delete(ctx, &DeleteReq{
		PrivateKey: serde.MarshalPrivateKey(privKey),
	})
	return err
}

func (c *client) MainAddr() (inet256.Addr, error) {
	ctx := context.Background()
	res, err := c.adminClient.GetStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return inet256.Addr{}, err
	}
	return inet256.AddrFromBytes(res.LocalAddr), nil
}

func (c *client) TransportAddrs() ([]p2p.Addr, error) {
	ctx := context.Background()
	res, err := c.adminClient.GetStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	var ret []p2p.Addr
	for _, addr := range res.TransportAddrs {
		ret = append(ret, TransportAddr(addr))
	}
	return ret, nil
}

func (c *client) PeerStatus() ([]mesh256.PeerStatus, error) {
	ctx := context.Background()
	req, err := c.adminClient.GetStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return peerStatusFromProto(req.PeerStatus), nil
}

type TransportAddr string

func (a TransportAddr) MarshalText() ([]byte, error) {
	return []byte(a), nil
}

func (a TransportAddr) Key() string {
	return string(a)
}

func (a TransportAddr) String() string {
	return string(a)
}
