package landisco

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"go.brendoncarroll.net/exp/slices2"
	"go.brendoncarroll.net/p2p/s/multiswarm"
	"go.brendoncarroll.net/stdctx/logctx"
	"go.brendoncarroll.net/tai64"
	"go.uber.org/zap"
	"golang.org/x/net/ipv6"
	"golang.org/x/sync/errgroup"

	"go.inet256.org/inet256/pkg/inet256"
)

const (
	MulticastPort = 25632
)

// udp6MulticastAddr is the address of the multicast group
var udp6MulticastAddr = net.UDPAddr{IP: net.IPv6linklocalallnodes, Port: MulticastPort}

// newUDPMulticast returns a UDP conn configured for multicast on the interface
func newUDPMulticast(ifName string) (*net.UDPConn, error) {
	iface, err := net.InterfaceByName(ifName)
	if err != nil {
		return nil, err
	}
	// conn, err := net.ListenMulticastUDP("udp6", iface, udp6MulticastAddr)
	// if err != nil {
	// 	return nil, err
	// }
	conn, err := net.ListenUDP("udp6", &udp6MulticastAddr)
	if err != nil {
		return nil, err
	}
	ipc := ipv6.NewPacketConn(conn)
	if err := ipc.SetMulticastLoopback(true); err != nil {
		return nil, err
	}
	// if err := ipc.SetMulticastInterface(iface); err != nil {
	// 	return nil, err
	// }
	if err := ipc.JoinGroup(iface, &udp6MulticastAddr); err != nil {
		return nil, err
	}
	return conn, nil
}

// Bus is a message bus backed by IPv6 multicast
type Bus struct {
	ifaces         []string
	announcePeriod time.Duration

	conns []*net.UDPConn
	cf    context.CancelFunc
	eg    errgroup.Group
	mu    sync.Mutex
	subs  []chan<- Message
}

func NewBus(bgCtx context.Context, ifaces []string, announcePeriod time.Duration) (*Bus, error) {
	if announcePeriod <= 0 {
		return nil, fmt.Errorf("invalid announcePeriod %v", announcePeriod)
	}
	if len(ifaces) < 1 {
		return nil, errors.New("must provide at least one interface by name")
	}
	var conns []*net.UDPConn
	for _, name := range ifaces {
		conn, err := newUDPMulticast(name)
		if err != nil {
			return nil, err
		}
		conns = append(conns, conn)
	}
	ctx, cf := context.WithCancel(bgCtx)
	b := &Bus{
		ifaces:         ifaces,
		announcePeriod: announcePeriod,

		conns: conns,
		cf:    cf,
		subs:  []chan<- Message{},
	}
	for _, c := range b.conns {
		c := c
		b.eg.Go(func() error {
			return b.readLoop(ctx, c)
		})
	}
	return b, nil
}

func (s *Bus) Broadcast(ctx context.Context, msg Message) error {
	var retErr error
	for _, conn := range s.conns {
		if _, err := conn.WriteTo(msg, &udp6MulticastAddr); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}
	return retErr
}

func (s *Bus) Announce(ctx context.Context, now tai64.TAI64N, id inet256.Addr, addrs []multiswarm.Addr) error {
	var addrStrs []string
	for _, addr := range addrs {
		addrStrs = append(addrStrs, addr.String())
	}
	data, err := json.Marshal(Advertisement{
		Transports: addrStrs,
	})
	if err != nil {
		return err
	}
	msg := NewMessage(now, id, data)
	return s.Broadcast(ctx, msg)
}

// Subscribe will begin writing to the channel
func (s *Bus) Subscribe(ch chan<- Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs = append(s.subs, ch)
}

func (s *Bus) Unsubscribe(ch chan<- Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs = slices2.Filter(s.subs, func(x chan<- Message) bool {
		return x != ch
	})
}

func (b *Bus) Close() (retErr error) {
	// close connections
	for _, conn := range b.conns {
		err := conn.Close()
		retErr = errors.Join(retErr, err)
	}
	// shutdown background processes
	b.cf()
	retErr = b.eg.Wait()
	return retErr
}

func (b *Bus) String() string {
	return fmt.Sprintf("IPv6Bus{%v}", b.ifaces)
}

func (b *Bus) readLoop(ctx context.Context, conn *net.UDPConn) error {
	buf := make([]byte, 1<<16)
	for {
		n, sender, err := conn.ReadFrom(buf[:])
		if err != nil {
			return err
		}
		msg, err := ParseMessage(buf[:n])
		if err != nil {
			logctx.Warn(ctx, "while parsing message", zap.Error(err), zap.Any("src", sender), zap.Int("len", n))
		}

		// deliver to subscribers
		b.mu.Lock()
		for _, ch := range b.subs {
			select {
			case ch <- msg:
			default:
			}
		}
		b.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}
