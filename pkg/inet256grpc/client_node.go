package inet256grpc

import (
	"context"
	"errors"
	"net"
	sync "sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type node struct {
	inetClient INET256Client
	privKey    p2p.PrivateKey
	localAddr  inet256.Addr

	eg            errgroup.Group
	cf            context.CancelFunc
	recvHub       *netutil.TellHub
	mu            sync.Mutex
	sendMu        sync.Mutex
	connectClient INET256_ConnectClient
}

func newNode(inetClient INET256Client, privKey p2p.PrivateKey) (*node, error) {
	ctx, cf := context.WithCancel(context.Background())
	n := &node{
		inetClient: inetClient,
		privKey:    privKey,
		localAddr:  inet256.NewAddr(privKey.Public()),

		cf:      cf,
		recvHub: netutil.NewTellHub(),
	}
	_, err := n.connect(ctx)
	if err != nil {
		return nil, err
	}
	n.eg.Go(func() error {
		return runForever(ctx, func() error {
			defer func() {
				n.mu.Lock()
				defer n.mu.Unlock()
				n.connectClient = nil
			}()
			cc, err := n.connect(ctx)
			if err != nil {
				return err
			}
			for {
				msg, err := cc.Recv()
				if err != nil {
					return err
				}
				dg := msg.Datagram
				if err := n.recvHub.Deliver(ctx, p2p.Message[inet256.Addr]{
					Src:     inet256.AddrFromBytes(dg.Src),
					Dst:     inet256.AddrFromBytes(dg.Dst),
					Payload: dg.Payload,
				}); err != nil {
					return err
				}
			}
		})
	})
	return n, nil
}

func (n *node) Send(ctx context.Context, dst inet256.Addr, data []byte) error {
	cc, err := n.connect(ctx)
	if err != nil {
		return err
	}
	n.sendMu.Lock()
	defer n.sendMu.Unlock()
	return cc.Send(&ConnectMsg{
		Datagram: &Datagram{
			Dst:     dst[:],
			Payload: data[:],
		},
	})
}

func (n *node) Receive(ctx context.Context, fn func(inet256.Message)) error {
	return n.recvHub.Receive(ctx, func(x p2p.Message[inet256.Addr]) {
		fn(inet256.Message{
			Src:     x.Src,
			Dst:     x.Dst,
			Payload: x.Payload,
		})
	})
}

func (n *node) Close() error {
	n.cf()
	n.recvHub.CloseWithError(net.ErrClosed)
	n.eg.Wait()
	return nil
}

func (n *node) LocalAddr() inet256.Addr {
	return n.localAddr
}

func (n *node) PublicKey() inet256.PublicKey {
	return n.privKey.Public()
}

func (n *node) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	res, err := n.inetClient.FindAddr(ctx, &FindAddrReq{
		PrivateKey: serde.MarshalPrivateKey(n.privKey),
		Prefix:     prefix,
		Nbits:      uint32(nbits),
	})
	if err != nil {
		logrus.Error(err)
		return inet256.Addr{}, inet256.ErrNoAddrWithPrefix
	}
	if len(res.Addr) > 0 {
		return inet256.AddrFromBytes(res.Addr), nil
	}
	return inet256.Addr{}, inet256.ErrNoAddrWithPrefix
}

func (n *node) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	res, err := n.inetClient.LookupPublicKey(ctx, &LookupPublicKeyReq{
		PrivateKey: serde.MarshalPrivateKey(n.privKey),
		Target:     target[:],
	})
	if err != nil {
		return inet256.Addr{}, err
	}
	return inet256.ParsePublicKey(res.PublicKey)
}

func (n *node) MTU(ctx context.Context, target inet256.Addr) int {
	res, err := n.inetClient.MTU(ctx, &MTUReq{
		PrivateKey: serde.MarshalPrivateKey(n.privKey),
		Target:     target[:],
	})
	if err != nil {
		return inet256.MinMTU
	}
	return int(res.Mtu)
}

func (n *node) connect(ctx context.Context) (INET256_ConnectClient, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sendMu.Lock()
	defer n.sendMu.Unlock()
	if n.connectClient != nil {
		return n.connectClient, nil
	}
	cc, err := n.inetClient.Connect(ctx)
	if err != nil {
		return nil, err
	}
	privKeyBytes := serde.MarshalPrivateKey(n.privKey)
	if err := cc.Send(&ConnectMsg{
		Init: &ConnectInit{
			PrivateKey: privKeyBytes,
		},
	}); err != nil {
		return nil, err
	}
	msg, err := cc.Recv()
	if err != nil {
		return nil, err
	}
	if len(msg.Established) == 0 {
		return nil, errors.New("connect not established")
	}
	n.connectClient = cc
	return cc, nil
}

func runForever(ctx context.Context, fn func() error) error {
	for {
		if err := fn(); err != nil {
			if errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled {
				return err
			}
			time.Sleep(time.Second)
		} else {
			panic("function should not return without error")
		}
	}
}
