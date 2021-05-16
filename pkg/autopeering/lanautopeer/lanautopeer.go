package lanautopeer

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/inet256/inet256/pkg/autopeering"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type service struct {
	udpAddr net.UDPAddr
}

func New(addr net.UDPAddr) autopeering.Service {
	return &service{
		udpAddr: addr,
	}
}

func (s *service) Run(ctx context.Context, params autopeering.Params) error {
	conn, err := net.ListenMulticastUDP("udp", nil, &s.udpAddr)
	if err != nil {
		return err
	}
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.readLoop(ctx, params, conn)
	})
	eg.Go(func() error {
		return s.broadcastLoop(ctx, params, conn)
	})
	eg.Go(func() error {
		<-ctx.Done()
		return conn.Close()
	})
	return eg.Wait()
}

func (s *service) readLoop(ctx context.Context, params autopeering.Params, conn *net.UDPConn) error {
	buf := make([]byte, 1<<16)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}
		data := buf[:n]
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			logrus.Warnf("error parsing message")
			continue
		}
	}
}

func (s *service) broadcastLoop(ctx context.Context, params autopeering.Params, conn *net.UDPConn) error {
	const period = 60 * time.Second
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		if err := s.broadcastOnce(ctx, params, conn); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *service) broadcastOnce(ctx context.Context, params autopeering.Params, conn *net.UDPConn) error {
	msg := Message{
		INET256Addr: params.LocalAddr,
		Endpoints:   params.AddrSource(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	_, err = conn.Write(data)
	return err
}

type Message struct {
	INET256Addr inet256.Addr `json:"inet256_addr"`
	Endpoints   []string     `json:"endpoints"`
}
