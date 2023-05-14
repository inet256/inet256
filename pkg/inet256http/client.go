package inet256http

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"

	"github.com/brendoncarroll/go-p2p/s/swarmutil/retry"
	"github.com/brendoncarroll/stdctx/logctx"
	"golang.org/x/exp/slices"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256ipc"
	"github.com/inet256/inet256/pkg/serde"
)

var _ inet256.Service = &Client{}

type ClientOption = func(*Client)

func WithDialer(d net.Dialer) func(c *Client) {
	return func(c *Client) {
		c.dialer = d
	}
}

type Client struct {
	endpoint *url.URL
	dialer   net.Dialer
}

func NewClient(endpoint string, opts ...ClientOption) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	validSchemes := []string{"tcp", "unix"}
	if !slices.Contains(validSchemes, u.Scheme) {
		return nil, fmt.Errorf("url scheme must be one of %v. HAVE: %q", validSchemes, endpoint)
	}
	c := &Client{
		endpoint: u,
		dialer:   net.Dialer{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func (c *Client) Open(ctx context.Context, privKey inet256.PrivateKey, opts ...inet256.NodeOption) (inet256.Node, error) {
	pubKey := privKey.Public()
	id := inet256.NewAddr(pubKey)
	p := path.Join("/nodes/", id.String())
	reqData, err := json.Marshal(OpenReq{
		PrivateKey: serde.MarshalPrivateKey(privKey),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodConnect, p, bytes.NewReader(reqData))
	if err != nil {
		return nil, err
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, err
	}
	br := bufio.NewReader(conn)
	res, err := http.ReadResponse(br, req)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("inet256http: error opening node %v", res.Status)
	}
	logctx.Debugf(ctx, "client opened node %v", id)
	return newNodeClient(br, conn, conn, pubKey), nil
}

func (c *Client) Drop(ctx context.Context, privKey inet256.PrivateKey) error {
	pubKey := privKey.Public()
	id := inet256.NewAddr(pubKey)
	p := path.Join("/nodes/", id.Base64String())
	reqData, err := json.Marshal(DropReq{PrivateKey: serde.MarshalPrivateKey(privKey)})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, p, bytes.NewReader(reqData))
	if err != nil {
		return err
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := req.Write(conn); err != nil {
		return err
	}
	res, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("non-okay status: %v", res.Status)
	}
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	return retry.Retry(ctx, func() error {
		c, err := c.dial(ctx)
		if err != nil {
			return err
		}
		return c.Close()
	})
}

func (c *Client) dial(ctx context.Context) (net.Conn, error) {
	return c.dialer.DialContext(ctx, c.endpoint.Scheme, path.Join(c.endpoint.Host, c.endpoint.Path))
}

var _ inet256.Node = nodeClient{}

type nodeClient struct {
	c io.Closer
	*inet256ipc.NodeClient
}

func newNodeClient(r io.Reader, w io.Writer, c io.Closer, pubKey inet256.PublicKey) nodeClient {
	fr := inet256ipc.NewStreamFramer(r, w)
	return nodeClient{
		c:          c,
		NodeClient: inet256ipc.NewNodeClient(fr, pubKey),
	}
}

func (nc nodeClient) Close() error {
	nc.c.Close()
	nc.NodeClient.Close()
	return nil
}
