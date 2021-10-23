package inet256http

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"io"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/pkg/errors"
)

var _ inet256.Service = &Client{}

type Client struct {
	endpoint string
	hc       *http.Client
}

func NewClient(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		hc:       &http.Client{},
	}
}

func (c *Client) CreateNode(ctx context.Context, privateKey inet256.PrivateKey) (inet256.Node, error) {
	return newNode(c, privateKey), nil
}

func (c *Client) DeleteNode(ctx context.Context, privateKey inet256.PrivateKey) error {
	res, err := c.do(ctx, request{
		Method: http.MethodDelete,
		Path:   "node",
		Headers: map[string]string{
			PrivateKeyHeader: base64.URLEncoding.EncodeToString(serde.MarshalPrivateKey(privateKey)),
		},
	})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func (c *Client) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	res, err := c.do(ctx, request{
		Method: http.MethodPost,
		Path:   "findAddr",
		Body: marshalJSON(FindAddrReq{
			Prefix: prefix,
			Nbits:  nbits,
		}),
	})
	if err != nil {
		return inet256.Addr{}, err
	}
	var x FindAddrRes
	if err := readJSON(res.Body, &x); err != nil {
		return inet256.Addr{}, err
	}
	return x.Addr, nil
}

func (c *Client) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	res, err := c.do(ctx, request{
		Method: http.MethodGet,
		Path:   "public-key/" + target.String(),
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return inet256.ParsePublicKey(data)
}

func (c *Client) MTU(ctx context.Context, target inet256.Addr) int {
	res, err := c.do(ctx, request{
		Method: http.MethodGet,
		Path:   "mtu/" + target.String(),
	})
	if err != nil {
		return inet256.MinMTU
	}
	defer res.Body.Close()
	var x MTUResp
	if err := readJSON(res.Body, &x); err != nil {
		return inet256.MinMTU
	}
	return x.MTU
}

func (c *Client) GetNodeInfo(ctx context.Context, target inet256.Addr) (*NodeInfo, error) {
	res, err := c.do(ctx, request{
		Method: http.MethodGet,
		Path:   "node/" + target.String(),
	})
	if err != nil {
		return nil, err
	}
	var x NodeInfo
	if err := readJSON(res.Body, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

type request struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
}

func (c *Client) do(ctx context.Context, r request) (*http.Response, error) {
	u := path.Join(c.endpoint, r.Path)
	req, err := http.NewRequestWithContext(ctx, r.Method, u, bytes.NewReader(r.Body))
	if err != nil {
		panic(err)
	}
	for k, v := range r.Headers {
		req.Header.Set(k, v)
	}
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("non OK status: %v %v", res.StatusCode, res.Status)
	}
	return res, nil
}

type node struct {
	c    *Client
	conn *conn

	privateKey inet256.PrivateKey
	localAddr  inet256.Addr
}

func newNode(c *Client, privateKey p2p.PrivateKey) *node {
	return &node{
		c:          c,
		privateKey: privateKey,
		localAddr:  inet256.NewAddr(privateKey.Public()),
	}
}

func (n *node) Tell(ctx context.Context, dst inet256.Addr, data []byte) error {
	c, err := n.getConn(ctx)
	if err != nil {
		return err
	}
	bufw := bufio.NewWriter(c)
	if err := writeMessage(bufw, inet256.Message{
		Src:     n.localAddr,
		Dst:     dst,
		Payload: data,
	}); err != nil {
		return err
	}
	return bufw.Flush()
}

func (nd *node) Receive(ctx context.Context, fn func(inet256.Message)) error {
	var buf [inet256.MaxMTU]byte
	c, err := nd.getConn(ctx)
	if err != nil {
		return err
	}
	var src, dst inet256.Addr
	n, err := readMessage(c, &src, &dst, buf[:])
	if err != nil {
		return err
	}
	fn(inet256.Message{
		Src:     src,
		Dst:     dst,
		Payload: buf[:n],
	})
	return nil
}

func (n *node) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	return n.c.LookupPublicKey(ctx, target)
}

func (n *node) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	return n.c.FindAddr(ctx, prefix, nbits)
}

func (n *node) Bootstrap(ctx context.Context) error {
	return nil
}

func (n *node) MTU(ctx context.Context, target inet256.Addr) int {
	return n.c.MTU(ctx, target)
}

func (n *node) LocalAddr() inet256.Addr {
	return n.localAddr
}

func (n *node) PublicKey() inet256.PublicKey {
	return n.privateKey.Public()
}

func (n *node) Close() error {
	if n.conn != nil {
		return n.conn.Close()
	}
	return nil
}

func (n *node) getConn(ctx context.Context) (io.ReadWriteCloser, error) {
	if n.conn != nil {
		return n.conn, nil
	}
	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodPost, path.Join(n.c.endpoint, "connect"), pr)
	if err != nil {
		panic(err)
	}
	setPrivateKey(req, n.privateKey)
	res, err := n.c.hc.Do(req)
	if err != nil {
		pw.Close()
		pr.Close()
		return nil, err
	}
	n.conn = &conn{incoming: res.Body, outgoing: pw}
	return n.conn, nil
}

func writeMessage(w io.Writer, msg inet256.Message) error {
	if err := binary.Write(w, binary.BigEndian, uint32(32+32+len(msg.Payload))); err != nil {
		return err
	}
	if _, err := w.Write(msg.Src[:]); err != nil {
		return err
	}
	if _, err := w.Write(msg.Dst[:]); err != nil {
		return err
	}
	if _, err := w.Write(msg.Payload[:]); err != nil {
		return err
	}
	return nil
}

func readMessage(r io.Reader, src, dst *inet256.Addr, buf []byte) (int, error) {
	var msgLen uint32
	if err := binary.Read(r, binary.BigEndian, msgLen); err != nil {
		return 0, err
	}
	if _, err := io.ReadFull(r, src[:]); err != nil {
		return 0, err
	}
	if _, err := io.ReadFull(r, dst[:]); err != nil {
		return 0, err
	}
	r = io.LimitReader(r, int64(msgLen-64))
	return r.Read(buf)
}

type conn struct {
	outgoing io.WriteCloser
	incoming io.ReadCloser
}

func (c conn) Write(p []byte) (int, error) {
	return c.outgoing.Write(p)
}

func (c conn) Read(p []byte) (int, error) {
	return c.incoming.Read(p)
}

func (c conn) Close() error {
	c.incoming.Close()
	c.outgoing.Close()
	return nil
}
