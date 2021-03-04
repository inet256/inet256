package inet256ipv6

import (
	"context"
	"net"
	"os/exec"
	"runtime"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
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

	if err := configureInterface(devName, localIPv6); err != nil {
		return err
	}
	p := portal{
		log:     log,
		af:      af,
		network: params.Network,
		dev:     dev,
	}
	return p.run(ctx)
}

type portal struct {
	log *logrus.Logger
	af  AllowFunc

	network inet256.Network
	dev     tun.Device
}

func (p *portal) run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return p.outboundLoop(ctx)
	})
	eg.Go(func() error {
		return p.inboundLoop(ctx)
	})
	eg.Go(func() error {
		for {
			select {
			case e := <-p.dev.Events():
				switch e {
				case tun.EventUp:
					p.log.Info("device up")
				case tun.EventDown:
					p.log.Info("device down")
				case tun.EventMTUUpdate:
					p.log.Info("mtu update")
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	return eg.Wait()
}

func (p *portal) outboundLoop(ctx context.Context) error {
	dev := p.dev
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
		data := buf[tunOffset:n]
		if err := p.handleOutbound(ctx, data); err != nil {
			p.log.Warn("skipping packet: ", err)
		}
	}
}

func (p *portal) handleOutbound(ctx context.Context, data []byte) error {
	network := p.network
	header, err := ipv6.ParseHeader(data)
	if err != nil {
		return err
	}
	prefix, nbits, err := IPv6ToPrefix(header.Dst)
	if err != nil {
		return err
	}
	dst, err := network.FindAddr(ctx, prefix, nbits)
	if err != nil {
		return err
	}
	return network.Tell(ctx, dst, data)
}

func (p *portal) inboundLoop(ctx context.Context) error {
	p.network.OnRecv(func(src, dst inet256.Addr, data []byte) {
		if !p.af(src) {
			return
		}
		if err := p.handleInbound(src, data); err != nil {
			p.log.Warn("ignoring INET256 message: ", err)
		}
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *portal) handleInbound(src inet256.Addr, data []byte) error {
	header, err := ipv6.ParseHeader(data)
	if err != nil {
		return err
	}
	srcIP := INet256ToIPv6(src)
	if !header.Src.Equal(srcIP) {
		return errors.Errorf("dropping inbound message from wrong source")
	}
	_, err = p.dev.Write(data, tunOffset)
	return err
}

func configureInterface(iface string, ipAddr net.IP) error {
	switch runtime.GOOS {
	case "darwin":
		return ifconfigCmd(iface, "inet6", ipAddr.String(), "prefixlen", "7")
	default:
		return errors.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func ifconfigCmd(args ...string) error {
	cmd := exec.Command("ifconfig", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	logrus.Info("ifconfig output:", string(output))
	return nil
}
