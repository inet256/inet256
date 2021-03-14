package inet256client

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type client struct {
	inetClient inet256grpc.INET256Client
	log        *logrus.Logger
}

func NewClient(endpoint string) (inet256.Service, error) {
	inetClient, err := dial(endpoint)
	if err != nil {
		return nil, err
	}
	return &client{inetClient: inetClient}, nil
}

func dial(endpoint string) (inet256grpc.INET256Client, error) {
	gc, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	inetClient := inet256grpc.NewINET256Client(gc)
	return inetClient, nil
}

func (c *client) CreateNode(ctx context.Context, privKey p2p.PrivateKey) (inet256.Node, error) {
	n, err := newNode(c.inetClient, privKey)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (c *client) DeleteNode(privKey p2p.PrivateKey) error {
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

func (c *client) MainAddr() inet256.Addr {
	ctx := context.Background()
	res, err := c.inetClient.GetStatus(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error(err)
		return inet256.Addr{}
	}
	return inet256.AddrFromBytes(res.LocalAddr)
}

func (c *client) TransportAddrs() []string {
	ctx := context.Background()
	res, err := c.inetClient.GetStatus(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error(err)
		return nil
	}
	return res.GetTransportAddrs()
}

func (c *client) PeerStatus() []inet256.PeerStatus {
	ctx := context.Background()
	req, err := c.inetClient.GetStatus(ctx, &emptypb.Empty{})
	if err != nil {
		c.log.Error(err)
		return nil
	}
	return inet256grpc.PeerStatusFromProto(req.PeerStatus)
}
