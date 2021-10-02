package inet256http

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
)

var _ inet256.Service = &Client{}

type Client struct {
	hc *http.Client
}

func (c *Client) CreateNode(ctx context.Context, privateKey inet256.PrivateKey) (inet256.Node, error) {
	return &node{
		c: c,
	}, nil
}

func (c *Client) DeleteNode(privateKey inet256.PrivateKey) error {
	panic("")
}

func (c *Client) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	panic("")
}

func (c *Client) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	panic("")
}

func (c *Client) MTU(ctx context.Context, target inet256.Addr) int {
	panic("")
}

type node struct {
	c    *Client
	conn net.Conn

	privateKey inet256.PrivateKey
	localAddr  inet256.Addr
	hub        *netutil.TellHub
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

func (n *node) ListOneHop() []inet256.Addr {
	return nil
}

func (n *node) Close() error {
	if n.conn != nil {
		return n.conn.Close()
	}
	return nil
}

func (n *node) getConn(ctx context.Context) (net.Conn, error) {
	if n.conn != nil {
		return n.conn, nil
	}
	panic("")
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
