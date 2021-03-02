package inet256ipv6

import (
	"context"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/ipv6"
	"golang.org/x/sync/errgroup"
	"golang.zx2c4.com/wireguard/tun"
)

const tunOffset = 4

type AllowFunc = func(inet256.Addr) bool

func AllowAll(inet256.Addr) bool {
	return true
}

type PortalParams struct {
	Network   inet256.Network
	AllowFunc AllowFunc
	Logger    *logrus.Logger
}

func RunPortal(ctx context.Context, params PortalParams) error {
	log := params.Logger
	af := params.AllowFunc
	if af == nil {
		af = AllowAll
	}
	dev, err := tun.CreateTUN("utun", 1<<15)
	if err != nil {
		return err
	}
	defer func() {
		if err := dev.Close(); err != nil {
			logrus.Error("error closing: ", err)
		}
	}()
	devName, err := dev.Name()
	if err != nil {
		return err
	}

	localAddr := params.Network.LocalAddr()
	localIPv6 := INet256ToIPv6(localAddr)
	log.Infof("Created TUN %s", devName)
	log.Infof("Local INET256: %v", localAddr)
	log.Infof("Local IPv6: %v", localIPv6)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return outboundLoop(ctx, params.Network, dev)
	})
	eg.Go(func() error {
		return inboundLoop(ctx, params.Network, dev, af)
	})
	eg.Go(func() error {
		for {
			select {
			case e := <-dev.Events():
				switch e {
				case tun.EventUp:
					log.Info("device up")
				case tun.EventDown:
					log.Info("device down")
				case tun.EventMTUUpdate:
					log.Info("mtu update")
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	return eg.Wait()
}

func outboundLoop(ctx context.Context, network inet256.Network, dev tun.Device) error {
	mtu, err := dev.MTU()
	if err != nil {
		return err
	}
	buf := make([]byte, mtu)
	for {
		n, err := dev.Read(buf, tunOffset)
		if err != nil {
			return err
		}
		data := buf[:n]
		if err := handleOutbound(ctx, network, data); err != nil {
			logrus.Warn("skipping packet: ", err)
		}
	}
}

func handleOutbound(ctx context.Context, network inet256.Network, data []byte) error {
	header, err := ipv6.ParseHeader(data)
	if err != nil {
		return err
	}
	prefix, nbits := IPv6ToPrefix(header.Dst)
	dst, err := network.FindAddr(ctx, prefix, nbits)
	if err != nil {
		return err
	}
	return network.Tell(ctx, dst, data)
}

func inboundLoop(ctx context.Context, n inet256.Network, dev tun.Device, af AllowFunc) error {
	n.OnRecv(func(src, dst inet256.Addr, data []byte) {
		if !af(src) {
			return
		}
		_, err := ipv6.ParseHeader(data)
		if err != nil {
			logrus.Warn("ignoring INET256 message: ", err)
		}
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}
