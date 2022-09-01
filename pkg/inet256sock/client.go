package inet256sock

import (
	"context"
	"fmt"
	"net"

	"github.com/brendoncarroll/go-p2p"
	"golang.org/x/crypto/sha3"
	"golang.org/x/sync/errgroup"

	"github.com/inet256/inet256/pkg/futures"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
)

var _ inet256.Node = &Client{}

type Client struct {
	localPubKey inet256.PublicKey
	localAddr   inet256.Addr
	uconn       *net.UnixConn

	tellHub    netutil.TellHub
	mtus       *futures.Store[[16]byte, int]
	findAddrs  *futures.Store[[16]byte, inet256.Addr]
	lookupPubs *futures.Store[[16]byte, inet256.PublicKey]

	eg errgroup.Group
}

// NewClient returns a client which will read and write from uconn in the background
func NewClient(uconn *net.UnixConn, localPubKey inet256.PublicKey) *Client {
	c := &Client{
		localPubKey: localPubKey,
		localAddr:   inet256.NewAddr(localPubKey),
		uconn:       uconn,

		tellHub:    *netutil.NewTellHub(),
		findAddrs:  futures.NewStore[[16]byte, inet256.Addr](),
		lookupPubs: futures.NewStore[[16]byte, inet256.PublicKey](),
		mtus:       futures.NewStore[[16]byte, int](),
	}
	// TODO: need context for lifecycle management
	c.eg.Go(func() error {
		return c.readLoop(context.Background())
	})
	return c
}

func (c *Client) readLoop(ctx context.Context) error {
	buf := make([]byte, 32+inet256.MaxMTU)
	for {
		n, _, err := c.uconn.ReadFrom(buf)
		if err != nil {
			return err
		}
		if err := c.handleMessage(ctx, buf[:n]); err != nil {
			continue
		}
	}
}

func (c *Client) handleMessage(ctx context.Context, x []byte) error {
	msg, err := AsMessage(x)
	if err != nil {
		return err
	}
	mt := msg.GetType()
	switch mt {
	case MT_Data:
		c.tellHub.Deliver(ctx, p2p.Message[inet256.Addr]{
			Src:     msg.GetDataAddr(),
			Dst:     c.localAddr,
			Payload: msg.GetPayload(),
		})
	case MT_FindAddrRes:
		if fut := c.findAddrs.Get(msg.GetRequestID()); fut != nil {
			fut.Succeed(msg.GetFindAddrRes())
		}
	case MT_PublicKeyRes:
		if fut := c.lookupPubs.Get(msg.GetRequestID()); fut != nil {
			pub, err := msg.GetPublicKeyRes()
			if err != nil {
				return err
			}
			fut.Succeed(pub)
		}
	case MT_MTURes:
		if fut := c.mtus.Get(msg.GetRequestID()); fut != nil {
			fut.Succeed(msg.GetMTURes())
		}
	default:
		return fmt.Errorf("not expecting type %v", mt)
	}
	return nil
}

func (c *Client) Receive(ctx context.Context, fn inet256.ReceiveFunc) error {
	return c.tellHub.Receive(ctx, func(x p2p.Message[inet256.Addr]) {
		fn(inet256.Message{
			Src:     x.Src,
			Dst:     x.Dst,
			Payload: x.Payload,
		})
	})
	return nil
}

func (c *Client) Send(ctx context.Context, dst inet256.Addr, data []byte) error {
	_, err := c.uconn.Write(MakeDataMessage(nil, dst, data))
	return err
}

func (c *Client) Close() error {
	return c.eg.Wait()
}

func (c *Client) LocalAddr() inet256.Addr {
	return c.localAddr
}

func (c *Client) PublicKey() inet256.PublicKey {
	return c.localPubKey
}

func (c *Client) MTU(ctx context.Context, target inet256.Addr) int {
	var msg Message

	reqID := NewRequestID(msg)
	fut, created := c.mtus.GetOrCreate(reqID)
	if !created {
		defer c.mtus.Delete(reqID)
	}
	if _, err := c.uconn.Write(msg); err != nil {
		return inet256.MinMTU
	}
	mtu, err := fut.Await(ctx)
	if err != nil {
		return inet256.MinMTU
	}
	return mtu
}

func (c *Client) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	var msg Message

	reqID := NewRequestID(msg)
	fut, created := c.findAddrs.GetOrCreate(reqID)
	if !created {
		defer c.findAddrs.Delete(reqID)
	}
	if _, err := c.uconn.Write(msg); err != nil {
		return inet256.Addr{}, err
	}
	return fut.Await(ctx)
}

func (c *Client) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	var msg Message

	var reqID [16]byte
	sha3.ShakeSum256(reqID[:], msg)
	fut, created := c.lookupPubs.GetOrCreate(reqID)
	if !created {
		defer c.lookupPubs.Delete(reqID)
	}
	if _, err := c.uconn.Write(msg); err != nil {
		return nil, err
	}
	return fut.Await(ctx)
}
