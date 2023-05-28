package landisco

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-tai64"
	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/peers"
	"github.com/inet256/inet256/pkg/serde"
	"go.uber.org/zap"
	"golang.org/x/net/ipv6"
	"golang.org/x/sync/errgroup"
)

const (
	MulticastPort = 25632
)

// udp6MulticastAddr is the address of the multicast group
var udp6MulticastAddr = net.UDPAddr{IP: net.IPv6linklocalallnodes, Port: MulticastPort}

type Service struct {
	ifaces []string
	conns  []*net.UDPConn
}

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

func New(ifaces []string) (*Service, error) {
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
	return &Service{ifaces: ifaces, conns: conns}, nil
}

func (s *Service) Announce(ctx context.Context, id inet256.Addr, addrs []multiswarm.Addr) error {
	var addrStrs []string
	for _, addr := range addrs {
		addrStrs = append(addrStrs, addr.String())
	}
	return s.announce(ctx, id, addrStrs)
}

func (s *Service) announce(ctx context.Context, id inet256.Addr, addrs []string) error {
	data, err := json.Marshal(Advertisement{
		Transports: addrs,
	})
	if err != nil {
		return err
	}
	msg := NewMessage(tai64.Now(), id, data)
	var retErr error
	for _, conn := range s.conns {
		if _, err := conn.WriteTo(msg, &udp6MulticastAddr); err != nil {
			retErr = errors.Join(retErr, err)
		}
	}
	return retErr
}

func (s *Service) Close() (retErr error) {
	for _, conn := range s.conns {
		err := conn.Close()
		retErr = errors.Join(retErr, err)
	}
	return retErr
}

func (s *Service) String() string {
	return fmt.Sprintf("LANDisco{%v}", s.ifaces)
}

func (s *Service) RunAddrDiscovery(ctx context.Context, params discovery.AddrDiscoveryParams) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.announceLoop(ctx, params)
	})
	for _, conn := range s.conns {
		conn := conn
		eg.Go(func() error {
			return s.readLoop(ctx, params, conn)
		})
	}
	return eg.Wait()
}

func (s *Service) announceLoop(ctx context.Context, params discovery.AddrDiscoveryParams) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.Announce(ctx, params.LocalID, params.GetLocalAddrs()); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) readLoop(ctx context.Context, params discovery.AddrDiscoveryParams, conn *net.UDPConn) error {
	buf := make([]byte, 1<<16)
	for {
		n, sender, err := conn.ReadFrom(buf[:])
		if err != nil {
			return err
		}
		_, err = s.handleMessage(params, buf[:n])
		if err != nil {
			logctx.Warn(ctx, "while handling message", zap.Error(err), zap.Any("src", sender), zap.Int("len", n))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

func (s *Service) handleMessage(params discovery.AddrDiscoveryParams, buf []byte) (bool, error) {
	msg, err := ParseMessage(buf)
	if err != nil {
		return false, err
	}
	// ignore self messages
	if _, _, err := msg.Open([]inet256.ID{params.LocalID}); err == nil {
		return false, nil
	}
	ps := params.AddressBook.List()
	i, data, err := msg.Open(ps)
	if err != nil {
		return false, err
	}
	adv := Advertisement{}
	if err := json.Unmarshal(data, &adv); err != nil {
		return false, err
	}
	addrs, err := serde.ParseAddrs(params.AddrParser, adv.Transports)
	if err != nil {
		return false, err
	}
	peers.SetAddrs[discovery.TransportAddr](params.AddressBook, ps[i], addrs)
	return adv.Solicit, nil
}
