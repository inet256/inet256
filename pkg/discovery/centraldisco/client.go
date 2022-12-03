package centraldisco

import (
	"context"
	"math"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-tai64"
	"github.com/inet256/inet256/pkg/discovery/centraldisco/internal"
	"github.com/inet256/inet256/pkg/inet256"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const purposeAnnounce = "inet256/discovery/central/announce"

type Client struct {
	client internal.DiscoveryClient
}

// NewClient returns a client connected using gc, and privatKey for signing announcements
func NewClient(gc grpc.ClientConnInterface) *Client {
	return &Client{
		client: internal.NewDiscoveryClient(gc),
	}
}

func (c *Client) Announce(ctx context.Context, privKey inet256.PrivateKey, endpoints []string, ttl time.Duration) error {
	ts := tai64.Now().TAI64().Marshal()
	announceBytes, _ := proto.Marshal(&internal.Announce{
		Endpoints:  endpoints,
		Tai64:      ts[:],
		TtlSeconds: int64(math.Ceil(ttl.Seconds())),
	})
	sig, err := p2p.Sign(nil, privKey.BuiltIn(), purposeAnnounce, announceBytes)
	if err != nil {
		return err
	}
	pubKeyBytes := inet256.MarshalPublicKey(nil, privKey.Public())
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
	switch {
	case status.Code(err) == codes.NotFound:
		return nil, nil
	case err != nil:
		return nil, err
	}
	pubKey, err := inet256.ParsePublicKey(res.PublicKey)
	if err != nil {
		return nil, err
	}
	if err := p2p.Verify(pubKey.BuiltIn(), purposeAnnounce, res.Announce, res.Sig); err != nil {
		return nil, err
	}
	var x internal.Announce
	if err := proto.Unmarshal(res.Announce, &x); err != nil {
		return nil, err
	}
	return x.Endpoints, nil
}
