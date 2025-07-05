package centraldisco

import (
	"context"
	"errors"
	"math"
	"time"

	"go.brendoncarroll.net/tai64"
	"go.inet256.org/inet256/src/discovery/centraldisco/internal"
	"go.inet256.org/inet256/src/inet256"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var purposeAnnounce = inet256.SigCtxString("inet256/discovery/central/announce")

type Client struct {
	pki    inet256.PKI
	client internal.DiscoveryClient
}

// NewClient returns a client connected using gc, and privatKey for signing announcements
func NewClient(gc grpc.ClientConnInterface) *Client {
	return &Client{
		pki:    inet256.DefaultPKI,
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
	sig := c.pki.Sign(&purposeAnnounce, privKey, announceBytes, nil)
	pubKeyBytes := inet256.MarshalPublicKey(nil, privKey.Public().(inet256.PublicKey))
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
	if !c.pki.Verify(&purposeAnnounce, pubKey, res.Announce, res.Sig) {
		return nil, errors.New("find: invalid signature for announce")
	}
	var x internal.Announce
	if err := proto.Unmarshal(res.Announce, &x); err != nil {
		return nil, err
	}
	return x.Endpoints, nil
}
