package inet256client

import (
	"context"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var _ inet256.Network = &Client{}

type Client struct {
	inetClient inet256grpc.INET256Client
	privKey    p2p.PrivateKey
	localAddr  inet256.Addr

	mu     sync.RWMutex
	onRecv inet256.RecvFunc
	cc     inet256grpc.INET256_ConnectClient
	cf     context.CancelFunc
}

func New(endpoint string, privKey p2p.PrivateKey) (*Client, error) {
	gc, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	inetClient := inet256grpc.NewINET256Client(gc)
	ctx, cf := context.WithCancel(context.Background())
	c := &Client{
		cf:         cf,
		inetClient: inetClient,
		privKey:    privKey,
		localAddr:  p2p.NewPeerID(privKey.Public()),
		onRecv:     inet256.NoOpRecvFunc,
	}
	go c.runLoop(ctx)
	return c, nil
}

func (c *Client) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	peerInfo, err := c.inetClient.Lookup(ctx, &inet256grpc.LookupReq{
		TargetAddr: prefix[:nbits/8],
	})
	if err != nil {
		return inet256.Addr{}, err
	}
	ret := inet256.Addr{}
	copy(ret[:], peerInfo.Addr)
	return ret, nil
}

func (c *Client) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	res, err := c.inetClient.Lookup(ctx, &inet256grpc.LookupReq{
		TargetAddr: target[:],
	})
	if err != nil {
		return nil, err
	}
	return inet256.ParsePublicKey(res.PublicKey)
}

func (c *Client) Tell(ctx context.Context, dst inet256.Addr, data []byte) error {
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

func (c *Client) OnRecv(fn inet256.RecvFunc) {
	if fn == nil {
		fn = inet256.NoOpRecvFunc
	}
	c.mu.Lock()
	c.mu.Unlock()
	c.onRecv = fn
}

func (c *Client) Close() error {
	c.cf()
	return nil
}

func (c *Client) MTU(ctx context.Context, target inet256.Addr) int {
	res, err := c.inetClient.MTU(ctx, &inet256grpc.MTUReq{
		Target: target[:],
	})
	if err != nil {
		return -1
	}
	return int(res.Mtu)
}

func (c *Client) LocalAddr() inet256.Addr {
	return c.localAddr
}

func (c *Client) runLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		cc, err := c.inetClient.Connect(ctx)
		if err != nil {
			logrus.Error(err)
		} else {
			if err := c.runClient(cc); err != nil {
				logrus.Error(err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Client) runClient(cc inet256grpc.INET256_ConnectClient) error {
	privKeyBytes, err := inet256.MarshalPrivateKey(c.privKey)
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
		src := inet256.AddrFromBytes(dg.Src)
		dst := inet256.AddrFromBytes(dg.Dst)
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
