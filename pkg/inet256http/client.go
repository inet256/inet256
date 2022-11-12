package inet256http

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"

	"github.com/brendoncarroll/go-p2p/s/swarmutil/retry"
	"github.com/brendoncarroll/stdctx/logctx"

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
	endpoint string
	dialer   net.Dialer
}

func NewClient(endpoint string, opts ...ClientOption) *Client {
	c := &Client{
		endpoint: endpoint,
		dialer:   net.Dialer{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Open(ctx context.Context, privKey inet256.PrivateKey, opts ...inet256.NodeOption) (inet256.Node, error) {
	pubKey := privKey.Public()
	id := inet256.NewAddr(pubKey)
	u, err := c.getURL()
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, id.String())
	reqData, err := json.Marshal(OpenReq{
		PrivateKey: serde.MarshalPrivateKey(privKey),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodConnect, u.String(), bytes.NewReader(reqData))
	if err != nil {
		return nil, err
	}
	conn, err := c.dialer.DialContext(ctx, "tcp", u.Host)
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
		return nil, errors.New("inet256http: error opening node")
	}
	if br.Buffered() > 0 {
		return nil, errors.New("data still in buffer")
	}
	logctx.Debugf(ctx, "client opened node %v", id)
	return newNodeClient(conn, pubKey), nil
}

func (c *Client) Drop(ctx context.Context, privKey inet256.PrivateKey) error {
	u, err := url.Parse(c.endpoint)
	if err != nil {
		return err
	}
	pubKey := privKey.Public()
	id := inet256.NewAddr(pubKey)
	p := path.Join(c.endpoint, id.Base64String())
	reqData, err := json.Marshal(DropReq{PrivateKey: serde.MarshalPrivateKey(privKey)})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, p, bytes.NewReader(reqData))
	if err != nil {
		return err
	}
	conn, err := c.dialer.DialContext(ctx, "tcp", u.Host)
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
	u, err := c.getURL()
	if err != nil {
		return err
	}
	return retry.Retry(ctx, func() error {
		c, err := c.dialer.DialContext(ctx, "tcp", u.Host)
		if err != nil {
			return err
		}
		return c.Close()
	})
}

func (c *Client) getURL() (*url.URL, error) {
	return url.Parse(c.endpoint)
}

var _ inet256.Node = nodeClient{}

type nodeClient struct {
	rwc io.ReadWriteCloser
	*inet256ipc.NodeClient
}

func newNodeClient(rwc io.ReadWriteCloser, pubKey inet256.PublicKey) nodeClient {
	fr := inet256ipc.NewStreamFramer(rwc)
	return nodeClient{
		rwc:        rwc,
		NodeClient: inet256ipc.NewNodeClient(fr, pubKey),
	}
}

func (nc nodeClient) Close() error {
	nc.rwc.Close()
	nc.NodeClient.Close()
	return nil
}
