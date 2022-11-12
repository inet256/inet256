package inet256ipc

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/futures"
	"github.com/brendoncarroll/go-p2p/s/swarmutil"
	"github.com/brendoncarroll/stdctx/logctx"
	"golang.org/x/sync/errgroup"

	"github.com/inet256/inet256/pkg/inet256"
)

type clientConfig struct{}

type NodeClientOption func(*clientConfig)

var _ inet256.Node = &NodeClient{}

type NodeClient struct {
	framer      Framer
	localPubKey inet256.PublicKey
	localAddr   inet256.Addr

	tellHub    swarmutil.TellHub[inet256.Addr]
	mtus       *futures.Store[[16]byte, int]
	findAddrs  *futures.Store[[16]byte, inet256.Addr]
	lookupPubs *futures.Store[[16]byte, inet256.PublicKey]
	framePool  *framePool

	cf context.CancelFunc
	eg errgroup.Group
}

// NewNodeClient returns a client which will read and write from rwc in the background
func NewNodeClient(fr Framer, localPubKey inet256.PublicKey, opts ...NodeClientOption) *NodeClient {
	ctx := context.Background()
	ctx, cf := context.WithCancel(ctx)
	c := &NodeClient{
		framer:      fr,
		localPubKey: localPubKey,
		localAddr:   inet256.NewAddr(localPubKey),

		tellHub:    *swarmutil.NewTellHub[inet256.Addr](),
		findAddrs:  futures.NewStore[[16]byte, inet256.Addr](),
		lookupPubs: futures.NewStore[[16]byte, inet256.PublicKey](),
		mtus:       futures.NewStore[[16]byte, int](),
		framePool:  newFramePool(),

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
	for {
		select {
		case <-ticker.C:
			fr := c.framePool.Acquire()
			WriteKeepAlive(fr)
			if err := c.framer.WriteFrame(ctx, fr); err != nil {
				return err
			}
			c.framePool.Release(fr)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *NodeClient) readLoop(ctx context.Context) error {
	fr := c.framePool.Acquire()
	defer c.framePool.Release(fr)
	for {
		if err := c.framer.ReadFrame(ctx, fr); err != nil {
			return err
		}
		if err := c.handleMessage(ctx, fr.Body()); err != nil {
			logctx.Errorf(ctx, "handling message %v", err)
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
	case MT_MTU:
		if fut := c.mtus.Get(msg.GetRequestID()); fut != nil {
			if res, err := msg.MTURes(); err != nil {
				fut.Fail(err)
			} else {
				fut.Succeed(res.MTU)
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
	return nil
}

func (c *NodeClient) Send(ctx context.Context, dst inet256.Addr, data []byte) error {
	fr := c.framePool.Acquire()
	defer c.framePool.Release(fr)
	WriteDataMessage(fr, dst, data)
	return c.framer.WriteFrame(ctx, fr)
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
	fr := c.framePool.Acquire()
	defer c.framePool.Release(fr)
	WriteRequest(fr, *reqID, mtype, req)
	return c.framer.WriteFrame(ctx, fr)
}

func (c *NodeClient) MTU(ctx context.Context, target inet256.Addr) int {
	reqID := NewRequestID()
	fut, _ := c.mtus.GetOrCreate(reqID)
	defer c.mtus.Delete(reqID, fut)
	if err := c.sendRequest(ctx, &reqID, MT_MTU, MTUReq{Target: target}); err != nil {
		logctx.Errorln(ctx, err)
		return inet256.MinMTU
	}
	mtu, err := futures.Await[int](ctx, fut)
	if err != nil {
		return inet256.MinMTU
	}
	return mtu
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
