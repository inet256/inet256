package inet256grpc

import (
	context "context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func NewExtendedClient(endpoint string) (inet256srv.Service, error) {
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

func (c *client) Open(ctx context.Context, privKey p2p.PrivateKey) (inet256.Node, error) {
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

func (c *client) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	peerInfo, err := c.inetClient.FindAddr(ctx, &FindAddrReq{
		Prefix: prefix,
		Nbits:  uint32(nbits),
	})
	if err != nil {
		return inet256.Addr{}, err
	}
	ret := inet256.Addr{}
	copy(ret[:], peerInfo.Addr)
	return ret, nil
}

func (c *client) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	res, err := c.inetClient.LookupPublicKey(ctx, &LookupPublicKeyReq{
		Target: target[:],
	})
	if err != nil {
		return nil, err
	}
	return inet256.ParsePublicKey(res.PublicKey)
}

func (c *client) MTU(ctx context.Context, target inet256.Addr) int {
	res, err := c.inetClient.MTU(ctx, &MTUReq{
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
