package landisco

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/serde"
	"golang.org/x/sync/errgroup"
)

const multicastAddr = "[ff05::c]:25501"

type service struct {
	addr     *net.UDPAddr
	announce chan struct{}
}

func New() discovery.Service {
	gaddr, err := net.ResolveUDPAddr("udp6", multicastAddr)
	if err != nil {
		panic(err)
	}
	return &service{
		addr:     gaddr,
		announce: make(chan struct{}),
	}
}

func (s *service) Run(ctx context.Context, params discovery.Params) error {
	conn, err := net.ListenMulticastUDP("udp6", nil, s.addr)
	if err != nil {
		return err
	}
	params.Logger.Info("listening on", s.addr)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.announceLoop(ctx, params, conn)
	})
	eg.Go(func() error {
		return s.readLoop(ctx, params, conn)
	})
	return eg.Wait()
}

func (s *service) announceLoop(ctx context.Context, params discovery.Params, conn *net.UDPConn) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		adv := Advertisement{
			Transports: serde.MarshalAddrs(params.GetLocalAddrs()),
		}
		data, err := json.Marshal(adv)
		if err != nil {
			panic(err)
		}
		msg := NewMessage(params.LocalID, data)
		if _, err := conn.Write(msg); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		case <-s.announce:
		}
	}
}

func (s *service) forceAnnounce() {
	s.announce <- struct{}{}
}

func (s *service) readLoop(ctx context.Context, params discovery.Params, conn *net.UDPConn) error {
	buf := make([]byte, 1<<16)
	for {
		n, err := conn.Read(buf[:])
		if err != nil {
			return err
		}
		if err := s.handleMessage(params, buf[:n]); err != nil {
			params.Logger.WithError(err).Error("lan discovery while handling message")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

func (s *service) handleMessage(params discovery.Params, buf []byte) error {
	msg, err := ParseMessage(buf)
	if err != nil {
		return err
	}
	peers := params.AddressBook.ListPeers()
	i, data, err := UnpackMessage(msg, peers)
	if err != nil {
		return err
	}
	adv := Advertisement{}
	if err := json.Unmarshal(data, &adv); err != nil {
		return err
	}
	addrs, err := serde.ParseAddrs(params.AddrParser, adv.Transports)
	if err != nil {
		return err
	}
	params.AddressBook.SetAddrs(peers[i], addrs)
	return nil
}
