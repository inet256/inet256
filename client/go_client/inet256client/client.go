package inet256client

import (
	"context"
	mrand "math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var _ inet256.Node = &client{}

type client struct {
	inetClient inet256grpc.INET256Client
	privKey    p2p.PrivateKey
	localAddr  inet256.Addr
	cf         context.CancelFunc

	mu     sync.RWMutex
	onRecv inet256.RecvFunc

	workers []*worker
}

func NewNode(endpoint string, privKey p2p.PrivateKey) (inet256.Node, error) {
	gc, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	inetClient := inet256grpc.NewINET256Client(gc)
	return NewNodeFromGRPC(inetClient, privKey)
}

func NewNodeFromGRPC(inetClient inet256grpc.INET256Client, privKey p2p.PrivateKey) (inet256.Node, error) {
	ctx, cf := context.WithCancel(context.Background())
	c := &client{
		cf:         cf,
		inetClient: inetClient,
		privKey:    privKey,
		localAddr:  p2p.NewPeerID(privKey.Public()),
		onRecv:     inet256.NoOpRecvFunc,
		workers:    make([]*worker, runtime.GOMAXPROCS(0)),
	}
	onRecvFn := func(src, dst inet256.Addr, payload []byte) {
		c.mu.RLock()
		defer c.mu.RUnlock()
		c.onRecv(src, dst, payload)
	}
	for i := range c.workers {
		c.workers[i] = newWorker(c.connect, onRecvFn)
	}
	go c.runLoop(ctx)

	if err := c.workers[0].waitReady(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *client) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
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

func (c *client) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	res, err := c.inetClient.Lookup(ctx, &inet256grpc.LookupReq{
		TargetAddr: target[:],
	})
	if err != nil {
		return nil, err
	}
	return inet256.ParsePublicKey(res.PublicKey)
}

func (c *client) Tell(ctx context.Context, dst inet256.Addr, data []byte) error {
	return c.pickWorker().tell(ctx, dst, data)
}

func (c *client) OnRecv(fn inet256.RecvFunc) {
	if fn == nil {
		fn = inet256.NoOpRecvFunc
	}
	c.mu.Lock()
	c.mu.Unlock()
	c.onRecv = fn
}

func (c *client) Close() error {
	c.cf()
	return nil
}

func (c *client) MTU(ctx context.Context, target inet256.Addr) int {
	res, err := c.inetClient.MTU(ctx, &inet256grpc.MTUReq{
		Target: target[:],
	})
	if err != nil {
		return -1
	}
	return int(res.Mtu)
}

func (c *client) LocalAddr() inet256.Addr {
	return c.localAddr
}

func (c *client) TransportAddrs() []string {
	return nil
}

func (c *client) ListOneHop() []inet256.Addr {
	// TODO: return the main node?
	return nil
}

func (c *client) WaitReady(ctx context.Context) error {
	return nil
}

func (c *client) runLoop(ctx context.Context) {
	eg := errgroup.Group{}
	for _, w := range c.workers {
		w := w
		eg.Go(func() error {
			return w.run(ctx)
		})
	}
	if err := eg.Wait(); err != nil {
		logrus.Error(err)
	}
}

func (c *client) connect(ctx context.Context) (inet256grpc.INET256_ConnectClient, error) {
	cc, err := c.inetClient.Connect(ctx)
	if err != nil {
		return nil, err
	}
	privKeyBytes, err := inet256.MarshalPrivateKey(c.privKey)
	if err != nil {
		panic(err)
	}
	if err := cc.Send(&inet256grpc.ConnectMsg{
		ConnectInit: &inet256grpc.ConnectInit{
			PrivateKey: privKeyBytes,
		},
	}); err != nil {
		return nil, err
	}
	return cc, nil
}

func (c *client) pickWorker() *worker {
	l := len(c.workers)
	offset := mrand.Intn(l)
	for i := range c.workers {
		i := (i + offset) % l
		if c.workers[i].isReady() {
			return c.workers[i]
		}
	}
	return c.workers[0]
}

type worker struct {
	getCC  func(context.Context) (inet256grpc.INET256_ConnectClient, error)
	onRecv inet256.RecvFunc

	mu sync.RWMutex
	cc inet256grpc.INET256_ConnectClient
}

func newWorker(fn func(context.Context) (inet256grpc.INET256_ConnectClient, error), onRecv inet256.RecvFunc) *worker {
	return &worker{
		getCC:  fn,
		onRecv: onRecv,
	}
}

func (w *worker) run(ctx context.Context) error {
	return runForever(ctx, func() error {
		cc, err := w.getCC(ctx)
		if err != nil {
			return err
		}
		w.setClient(cc)
		for {
			msg, err := cc.Recv()
			if err != nil {
				return err
			}
			if msg.Datagram == nil {
				continue
			}
			dg := msg.Datagram
			src := inet256.AddrFromBytes(dg.Src)
			dst := inet256.AddrFromBytes(dg.Dst)
			w.onRecv(src, dst, dg.Payload)
		}
	})
}

func (w *worker) tell(ctx context.Context, dst inet256.Addr, data []byte) error {
	cc := w.getClient()
	if cc == nil {
		return errors.Errorf("cannot send, no active connection to daemon")
	}
	return cc.Send(&inet256grpc.ConnectMsg{
		Datagram: &inet256grpc.Datagram{
			Dst:     dst[:],
			Payload: data,
		},
	})
}

func (w *worker) setClient(x inet256grpc.INET256_ConnectClient) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cc = x
}

func (w *worker) getClient() inet256grpc.INET256_ConnectClient {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cc
}

func (w *worker) waitReady(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if w.isReady() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		default:
		}
	}
}

func (w *worker) isReady() bool {
	return w.getClient() != nil
}

func runForever(ctx context.Context, fn func() error) error {
	for {
		if err := fn(); err != nil {
			if err == context.Canceled {
				return err
			}
			time.Sleep(time.Second)
		} else {
			panic("function should not return without error")
		}
	}
}
