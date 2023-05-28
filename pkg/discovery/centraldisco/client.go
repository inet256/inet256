package centraldisco

import (
	"context"
	"errors"
	"math"
	"time"

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
	sig := inet256.Sign(nil, privKey, purposeAnnounce, announceBytes)
	pubKeyBytes := inet256.MarshalPublicKey(nil, privKey.Public())
	_, err := c.client.Announce(ctx, &internal.AnnounceReq{
		PublicKey: pubKeyBytes,
		Announce:  announceBytes,
		Sig:       sig,
	})
	return err
}

func (c *Client) Lookup(ctx context.Context, target inet256.Addr) ([]string, error) {
	res, err := c.client.Lookup(ctx, &internal.LookupReq{
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
	if !inet256.Verify(pubKey, purposeAnnounce, res.Announce, res.Sig) {
		return nil, errors.New("find: invalid signature for announce")
	}
	var x internal.Announce
	if err := proto.Unmarshal(res.Announce, &x); err != nil {
		return nil, err
	}
	return x.Endpoints, nil
}
