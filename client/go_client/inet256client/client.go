package inet256client

import (
	"context"
	"os"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const defaultAPIAddr = inet256d.DefaultAPIEndpoint

type client struct {
	inetClient  inet256grpc.INET256Client
	adminClient inet256grpc.AdminClient
	log         *logrus.Logger
}

func NewExtendedClient(endpoint string) (inet256srv.Service, error) {
	gc, err := dial(endpoint)
	if err != nil {
		return nil, err
	}
	return &client{
		inetClient:  inet256grpc.NewINET256Client(gc),
		adminClient: inet256grpc.NewAdminClient(gc),
		log:         logrus.StandardLogger(),
	}, nil
}

// NewClient creates an INET256 service using the specified endpoint for the API.
func NewClient(endpoint string) (inet256.Service, error) {
	return NewExtendedClient(endpoint)
}

// NewEnvClient creates an INET256 service using the environment variables to find the API.
// If you are looking for a inet256.Service constructor, this is probably the one you want.
// It checks the environment variable `INET256_API`
func NewEnvClient() (inet256.Service, error) {
	endpoint, yes := os.LookupEnv("INET256_API")
	if !yes {
		endpoint = defaultAPIAddr
	}
	return NewClient(endpoint)
}

func dial(endpoint string) (*grpc.ClientConn, error) {
	return grpc.Dial(endpoint, grpc.WithInsecure())
}

func (c *client) CreateNode(ctx context.Context, privKey p2p.PrivateKey) (inet256.Node, error) {
	n, err := newNode(c.inetClient, privKey)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (c *client) DeleteNode(ctx context.Context, privKey p2p.PrivateKey) error {
	panic("not implemented")
}

func (c *client) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	peerInfo, err := c.inetClient.Lookup(ctx, &inet256grpc.LookupReq{
		TargetAddr: prefix[:nbits/8],
	})
	if err != nil {
		return inet256.Addr{}, err
	}
	ret := inet256.Addr{}
	copy(ret[:], peerInfo.Addr)
	return ret, nil
}

func (c *client) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	res, err := c.inetClient.Lookup(ctx, &inet256grpc.LookupReq{
		TargetAddr: target[:],
	})
	if err != nil {
		return nil, err
	}
	return inet256.ParsePublicKey(res.PublicKey)
}

func (c *client) MTU(ctx context.Context, target inet256.Addr) int {
	res, err := c.inetClient.MTU(ctx, &inet256grpc.MTUReq{
		Target: target[:],
	})
	if err != nil {
		return -1
	}
	return int(res.Mtu)
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

func (c *client) PeerStatus() ([]inet256srv.PeerStatus, error) {
	ctx := context.Background()
	req, err := c.adminClient.GetStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return inet256grpc.PeerStatusFromProto(req.PeerStatus), nil
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
