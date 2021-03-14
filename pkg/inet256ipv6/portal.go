package inet256ipv6

import (
	"context"
	"net"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/ipv6"
	"golang.org/x/sync/errgroup"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	tunOffset     = 4
	cacheEntryTTL = 10 * time.Minute
)

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
	dev, err := tun.CreateTUN("utun", inet256.MinMTU)
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
	cache   sync.Map
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
	eg.Go(func() error {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			now := time.Now()
			p.cache.Range(func(k, v interface{}) bool {
				ent := v.(*cacheEntry)
				if now.Sub(ent.createdAt) > cacheEntryTTL {
					p.cache.Delete(k)
				}
				return true
			})
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
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
			p.log.Warn("outbound: dropping packet: ", err)
		}
	}
}

func (p *portal) handleOutbound(ctx context.Context, data []byte) error {
	network := p.network
	header, err := ipv6.ParseHeader(data)
	if err != nil {
		return err
	}
	var dst inet256.Addr
	if e := p.getEntry(IPv6FromBytes(header.Dst)); e != nil {
		dst = e.addr
	} else {
		prefix, nbits, err := IPv6ToPrefix(header.Dst)
		if err != nil {
			return err
		}
		dst, err = network.FindAddr(ctx, prefix, nbits)
		if err != nil {
			return err
		}
		ent := &cacheEntry{addr: dst, createdAt: time.Now()}
		p.putEntry(IPv6FromBytes(header.Dst), ent)
	}
	return network.Tell(ctx, dst, data)
}

func (p *portal) inboundLoop(ctx context.Context) error {
	return p.network.Recv(func(src, dst inet256.Addr, data []byte) {
		if !p.af(src) {
			return
		}
		if err := p.handleInbound(src, data); err != nil {
			p.log.Warn("inbound: ignoring INET256 message: ", err)
		}
	})
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
	_, err = p.dev.Write(data, 0)
	return err
}

func (p *portal) putEntry(key IPv6Addr, e *cacheEntry) {
	p.cache.LoadOrStore(key, e)
}

func (p *portal) getEntry(key IPv6Addr) *cacheEntry {
	v, ok := p.cache.Load(key)
	if !ok {
		return nil
	}
	return v.(*cacheEntry)
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

type cacheEntry struct {
	addr      inet256.Addr
	createdAt time.Time
}
