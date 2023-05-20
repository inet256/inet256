package inet256ipc

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/futures"
	"github.com/brendoncarroll/stdctx/logctx"
	"golang.org/x/sync/errgroup"

	"github.com/inet256/inet256/internal/netutil"
	"github.com/inet256/inet256/pkg/inet256"
)

type clientConfig struct{}

type NodeClientOption func(*clientConfig)

var _ inet256.Node = &NodeClient{}

type SendReceiver interface {
	Send(ctx context.Context, data []byte) error
	Receive(ctx context.Context, fn func(data []byte)) error
}

type NodeClient struct {
	transport   SendReceiver
	localPubKey inet256.PublicKey
	localAddr   inet256.Addr

	tellHub    netutil.TellHub
	mtus       *futures.Store[[16]byte, int]
	findAddrs  *futures.Store[[16]byte, inet256.Addr]
	lookupPubs *futures.Store[[16]byte, inet256.PublicKey]

	cf context.CancelFunc
	eg errgroup.Group
}

// NewNodeClient returns a client which will read and write from rwc in the background
func NewNodeClient(transport SendReceiver, localPubKey inet256.PublicKey, opts ...NodeClientOption) *NodeClient {
	ctx := context.Background()
	ctx, cf := context.WithCancel(ctx)
	c := &NodeClient{
		transport:   transport,
		localPubKey: localPubKey,
		localAddr:   inet256.NewAddr(localPubKey),

		tellHub:    netutil.NewTellHub(),
		findAddrs:  futures.NewStore[[16]byte, inet256.Addr](),
		lookupPubs: futures.NewStore[[16]byte, inet256.PublicKey](),
		mtus:       futures.NewStore[[16]byte, int](),

		cf: cf,
	}
	for i := 0; i < runtime.NumCPU(); i++ {
		c.eg.Go(func() error {
			return c.readLoop(ctx)
		})
	}
	c.eg.Go(func() error {
		return c.keepAliveLoop(ctx)
	})
	return c
}

func (c *NodeClient) keepAliveLoop(ctx context.Context) error {
	defer c.cf()
	ticker := time.NewTicker(DefaultKeepAliveInterval)
	defer ticker.Stop()
	var buf [MaxMessageLen]byte
	for {
		select {
		case <-ticker.C:
			n := WriteKeepAlive(buf[:])
			if err := c.transport.Send(ctx, buf[:n]); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *NodeClient) readLoop(ctx context.Context) error {
	for {
		if err := c.transport.Receive(ctx, func(data []byte) {
			if err := c.handleMessage(ctx, data); err != nil {
				logctx.Errorf(ctx, "handling message %v", err)
			}
		}); err != nil {
			return err
		}
	}
}

func (c *NodeClient) handleMessage(ctx context.Context, x []byte) error {
	msg, err := AsMessage(x, false)
	if err != nil {
		return err
	}
	mt := msg.GetType()
	switch mt {
	case MT_Data:
		dataMsg := msg.DataMsg()
		c.tellHub.Deliver(ctx, p2p.Message[inet256.Addr]{
			Src:     dataMsg.Addr,
			Dst:     c.localAddr,
			Payload: dataMsg.Payload,
		})
	case MT_FindAddr:
		if fut := c.findAddrs.Get(msg.GetRequestID()); fut != nil {
			if res, err := msg.FindAddrRes(); err != nil {
				fut.Fail(err)
			} else {
				fut.Succeed(res.Addr)
			}
		}
	case MT_PublicKey:
		if fut := c.lookupPubs.Get(msg.GetRequestID()); fut != nil {
			if res, err := msg.LookupPublicKeyRes(); err != nil {
				fut.Fail(err)
			} else {
				pubKey, err := inet256.ParsePublicKey(res.PublicKey)
				if err != nil {
					fut.Fail(err)
				} else {
					fut.Succeed(pubKey)
				}
			}
		}
	case MT_KeepAlive:
	default:
		return fmt.Errorf("not expecting type %v", mt)
	}
	return nil
}

func (c *NodeClient) Receive(ctx context.Context, fn inet256.ReceiveFunc) error {
	return c.tellHub.Receive(ctx, func(x p2p.Message[inet256.Addr]) {
		fn(inet256.Message{
			Src:     x.Src,
			Dst:     x.Dst,
			Payload: x.Payload,
		})
	})
}

func (c *NodeClient) Send(ctx context.Context, dst inet256.Addr, data []byte) error {
	var buf [MaxMessageLen]byte
	n := WriteDataMessage(buf[:], dst, data)
	return c.transport.Send(ctx, buf[:n])
}

func (c *NodeClient) Close() error {
	c.cf()
	return c.eg.Wait()
}

func (c *NodeClient) LocalAddr() inet256.Addr {
	return c.localAddr
}

func (c *NodeClient) PublicKey() inet256.PublicKey {
	return c.localPubKey
}

func (c *NodeClient) sendRequest(ctx context.Context, reqID *[16]byte, mtype MessageType, req any) error {
	var buf [MaxMessageLen]byte
	n := WriteRequest(buf[:], *reqID, mtype, req)
	return c.transport.Send(ctx, buf[:n])
}

func (c *NodeClient) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	reqID := NewRequestID()
	fut, _ := c.findAddrs.GetOrCreate(reqID)
	defer c.findAddrs.Delete(reqID, fut)
	if err := c.sendRequest(ctx, &reqID, MT_FindAddr, FindAddrReq{Prefix: prefix, Nbits: nbits}); err != nil {
		return inet256.Addr{}, err
	}
	return futures.Await[inet256.Addr](ctx, fut)
}

func (c *NodeClient) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	reqID := NewRequestID()
	fut, _ := c.lookupPubs.GetOrCreate(reqID)
	defer c.lookupPubs.Delete(reqID, fut)
	if err := c.sendRequest(ctx, &reqID, MT_PublicKey, LookupPublicKeyReq{Target: target}); err != nil {
		return nil, err
	}
	return futures.Await[inet256.PublicKey](ctx, fut)
}
