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
)

type node struct {
	inetClient inet256grpc.INET256Client
	privKey    p2p.PrivateKey
	localAddr  inet256.Addr
	cf         context.CancelFunc

	recvHub *inet256.TellHub
	workers []*worker
}

func NewNode(endpoint string, privKey p2p.PrivateKey) (inet256.Node, error) {
	c, err := dial(endpoint)
	if err != nil {
		return nil, err
	}
	return newNode(c, privKey)
}

func NewNodeFromGRPC(client inet256grpc.INET256Client, privKey p2p.PrivateKey) (inet256.Node, error) {
	return newNode(client, privKey)
}

func newNode(inetClient inet256grpc.INET256Client, privKey p2p.PrivateKey) (*node, error) {
	ctx, cf := context.WithCancel(context.Background())
	n := &node{
		cf:         cf,
		inetClient: inetClient,
		privKey:    privKey,
		localAddr:  p2p.NewPeerID(privKey.Public()),
		recvHub:    inet256.NewTellHub(),
		workers:    make([]*worker, runtime.GOMAXPROCS(0)),
	}
	for i := range n.workers {
		n.workers[i] = newWorker(n.connect, n.recvHub)
	}
	go n.runLoop(ctx)
	if err := n.workers[0].bootstrap(ctx); err != nil {
		return nil, err
	}
	return n, nil
}

func (n *node) Tell(ctx context.Context, dst inet256.Addr, data []byte) error {
	return n.pickWorker().tell(ctx, dst, data)
}

func (n *node) Recv(ctx context.Context, src, dst *inet256.Addr, buf []byte) (int, error) {
	return n.recvHub.Recv(ctx, src, dst, buf)
}

func (n *node) WaitRecv(ctx context.Context) error {
	return n.recvHub.Wait(ctx)
}

func (n *node) Close() error {
	n.cf()
	return nil
}

func (n *node) LocalAddr() inet256.Addr {
	return n.localAddr
}

func (n *node) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	return n.getClient().FindAddr(ctx, prefix, nbits)
}

func (n *node) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	return n.getClient().LookupPublicKey(ctx, target)
}

func (n *node) MTU(ctx context.Context, addr inet256.Addr) int {
	return n.getClient().MTU(ctx, addr)
}

func (n *node) Bootstrap(ctx context.Context) error {
	eg := errgroup.Group{}
	for i := range n.workers {
		w := n.workers[i]
		eg.Go(func() error { return w.bootstrap(ctx) })
	}
	return eg.Wait()
}

func (n *node) ListOneHop() []inet256.Addr {
	return nil
}

func (n *node) getClient() *client {
	return &client{inetClient: n.inetClient}
}

func (n *node) runLoop(ctx context.Context) {
	eg := errgroup.Group{}
	for _, w := range n.workers {
		w := w
		eg.Go(func() error {
			return w.run(ctx)
		})
	}
	if err := eg.Wait(); err != nil {
		logrus.Error(err)
	}
}

func (n *node) connect(ctx context.Context) (inet256grpc.INET256_ConnectClient, error) {
	cc, err := n.inetClient.Connect(ctx)
	if err != nil {
		return nil, err
	}
	privKeyBytes, err := inet256.MarshalPrivateKey(n.privKey)
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

func (n *node) pickWorker() *worker {
	l := len(n.workers)
	offset := mrand.Intn(l)
	for i := range n.workers {
		i := (i + offset) % l
		if n.workers[i].isReady() {
			return n.workers[i]
		}
	}
	return n.workers[0]
}

type worker struct {
	getCC func(context.Context) (inet256grpc.INET256_ConnectClient, error)
	tells *inet256.TellHub

	mu sync.RWMutex
	cc inet256grpc.INET256_ConnectClient
}

func newWorker(fn func(context.Context) (inet256grpc.INET256_ConnectClient, error), tells *inet256.TellHub) *worker {
	return &worker{
		getCC: fn,
		tells: tells,
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
			dg := msg.Datagram
			if err := w.tells.Deliver(ctx, inet256.Message{
				Src:     inet256.AddrFromBytes(dg.Src),
				Dst:     inet256.AddrFromBytes(dg.Dst),
				Payload: dg.Payload,
			}); err != nil {
				return err
			}
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

func (w *worker) bootstrap(ctx context.Context) error {
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
