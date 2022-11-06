package inet256grpc

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/serde"
)

func NewClient(endpoint string) (inet256.Service, error) {
	gc, err := dial(endpoint)
	if err != nil {
		return nil, err
	}
	return &client{
		inetClient: NewINET256Client(gc),
		log:        logrus.StandardLogger(),
	}, nil
}

func dial(endpoint string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append([]grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}, opts...)
	return grpc.Dial(endpoint, opts...)
}

type client struct {
	inetClient INET256Client
	log        *logrus.Logger
}

func (c *client) Open(ctx context.Context, privKey p2p.PrivateKey, opts ...inet256.NodeOption) (inet256.Node, error) {
	n, err := newNode(c.inetClient, privKey)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (c *client) Drop(ctx context.Context, privKey p2p.PrivateKey) error {
	_, err := c.inetClient.Drop(ctx, &DropReq{
		PrivateKey: serde.MarshalPrivateKey(privKey),
	})
	return err
}
