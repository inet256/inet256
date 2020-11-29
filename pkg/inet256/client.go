package inet256

import (
	"context"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ Network = &Client{}

type Client struct {
	gc      inet256grpc.INET256Client
	privKey p2p.PrivateKey

	mu     sync.RWMutex
	onRecv RecvFunc
	cc     inet256grpc.INET256_ConnectClient
	cf     context.CancelFunc
}

func NewClient(gc inet256grpc.INET256Client, privKey p2p.PrivateKey) *Client {
	ctx, cf := context.WithCancel(context.Background())
	c := &Client{
		cf:      cf,
		gc:      gc,
		privKey: privKey,
		onRecv:  NoOpRecvFunc,
	}
	go c.runLoop(ctx)
	return c
}

func (c *Client) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	panic("not implemented")
}

func (c *Client) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	res, err := c.gc.Lookup(ctx, &inet256grpc.LookupReq{
		TargetAddr: target[:],
	})
	if err != nil {
		return nil, err
	}
	return ParsePublicKey(res.PublicKey)
}

func (c *Client) Tell(ctx context.Context, dst Addr, data []byte) error {
	cc := c.getConnectClient()
	if cc == nil {
		return errors.Errorf("no ConnectClient")
	}
	return c.cc.Send(&inet256grpc.ConnectMsg{
		Datagram: &inet256grpc.Datagram{
			Dst:     dst[:],
			Payload: data,
		},
	})
}

func (c *Client) OnRecv(fn RecvFunc) {
	if fn == nil {
		fn = NoOpRecvFunc
	}
	c.mu.Lock()
	c.mu.Unlock()
	c.onRecv = fn
}

func (c *Client) Close() error {
	c.cf()
	return nil
}

func (c *Client) MTU(ctx context.Context, target Addr) int {
	res, err := c.gc.MTU(ctx, &inet256grpc.MTUReq{
		Target: target[:],
	})
	if err != nil {
		return -1
	}
	return int(res.Mtu)
}

func (c *Client) runLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		}
		cc, err := c.gc.Connect(ctx)
		if err != nil {
			logrus.Error(err)
			continue
		}
		if err := c.runClient(cc); err != nil {
			logrus.Error(err)
		}
	}
}

func (c *Client) runClient(cc inet256grpc.INET256_ConnectClient) error {
	privKeyBytes, err := MarshalPrivateKey(c.privKey)
	if err != nil {
		panic(err)
	}
	if err := cc.Send(&inet256grpc.ConnectMsg{
		ConnectInit: &inet256grpc.ConnectInit{
			PrivateKey: privKeyBytes,
		},
	}); err != nil {
		return err
	}
	c.setConnectClient(cc)
	defer c.setConnectClient(nil)
	for {
		msg, err := cc.Recv()
		if err != nil {
			return err
		}
		if msg.Datagram == nil {
			continue
		}
		dg := msg.Datagram
		c.mu.RLock()
		onRecv := c.onRecv
		c.mu.RUnlock()
		src := AddrFromBytes(dg.Src)
		dst := AddrFromBytes(dg.Dst)
		onRecv(src, dst, dg.Payload)
	}
}

func (c *Client) setConnectClient(cc inet256grpc.INET256_ConnectClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cc = cc
}

func (c *Client) getConnectClient() inet256grpc.INET256_ConnectClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cc
}
