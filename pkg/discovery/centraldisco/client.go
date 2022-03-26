package centraldisco

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-tai64"
	"github.com/inet256/inet256/pkg/discovery/centraldisco/internal"
	"github.com/inet256/inet256/pkg/inet256"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const purposeAnnounce = "inet256/centraldisco/announce"

type Client struct {
	client     internal.DiscoveryClient
	privateKey p2p.PrivateKey
}

func DialClient(ctx context.Context, addr string) (*Client, error) {
	gc, err := grpc.DialContext(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: internal.NewDiscoveryClient(gc),
	}, nil
}

func NewClient(gc grpc.ClientConnInterface, privateKey p2p.PrivateKey) *Client {
	return &Client{
		client:     internal.NewDiscoveryClient(gc),
		privateKey: privateKey,
	}
}

func (c *Client) Announce(ctx context.Context, endpoints []string) error {
	ts := tai64.Now().TAI64().Marshal()
	announceBytes, _ := proto.Marshal(&internal.Announce{
		Endpoints: endpoints,
		Tai64:     ts[:],
	})
	sig, err := p2p.Sign(nil, c.privateKey, purposeAnnounce, announceBytes)
	if err != nil {
		return err
	}
	pubKeyBytes := inet256.MarshalPublicKey(c.privateKey.Public())
	_, err = c.client.Announce(ctx, &internal.AnnounceReq{
		PublicKey: pubKeyBytes,
		Announce:  announceBytes,
		Sig:       sig,
	})
	return err
}

func (c *Client) Find(ctx context.Context, target inet256.Addr) ([]string, error) {
	res, err := c.client.Find(ctx, &internal.FindReq{
		Target: target[:],
	})
	if err != nil {
		return nil, err
	}
	pubKey, err := inet256.ParsePublicKey(res.PublicKey)
	if err != nil {
		return nil, err
	}
	if err := p2p.Verify(pubKey, purposeAnnounce, res.Announce, res.Sig); err != nil {
		return nil, err
	}
	var x internal.Announce
	if err := proto.Unmarshal(res.Announce, &x); err != nil {
		return nil, err
	}
	return x.Endpoints, nil
}
