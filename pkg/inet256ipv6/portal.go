package inet256ipv6

import (
	"context"
	"net/netip"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.brendoncarroll.net/stdctx/logctx"
	"golang.org/x/net/ipv6"
	"golang.org/x/sync/errgroup"
	"golang.zx2c4.com/wireguard/tun"

	"go.inet256.org/inet256/pkg/inet256"
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
	Node      inet256.Node
	AllowFunc AllowFunc
}

func RunPortal(ctx context.Context, params PortalParams) error {
	af := params.AllowFunc
	if af == nil {
		af = AllowAll
	}
	dev, err := tun.CreateTUN("utun", inet256.MTU)
	if err != nil {
		return err
	}
	defer func() {
		if err := dev.Close(); err != nil {
			logctx.Errorln(ctx, "error closing: ", err)
		}
	}()
	devName, err := dev.Name()
	if err != nil {
		return err
	}
	localAddr := params.Node.LocalAddr()
	localIPv6 := IPv6FromINET256(localAddr)
	logctx.Infof(ctx, "Created TUN %s", devName)
	logctx.Infof(ctx, "Local INET256: %v", localAddr)
	logctx.Infof(ctx, "Local IPv6: %v", localIPv6)

	if err := configureInterface(ctx, devName, localIPv6); err != nil {
		return err
	}
	p := portal{
		af:   af,
		node: params.Node,
		dev:  dev,
	}
	return p.run(ctx)
}

type portal struct {
	af AllowFunc

	node  inet256.Node
	dev   tun.Device
	cache sync.Map
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
					logctx.Infoln(ctx, "device up")
				case tun.EventDown:
					logctx.Infoln(ctx, "device down")
				case tun.EventMTUUpdate:
					logctx.Infoln(ctx, "mtu update")
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
			logctx.Warnln(ctx, "outbound: dropping packet: ", err)
		}
	}
}

func (p *portal) handleOutbound(ctx context.Context, data []byte) error {
	node := p.node
	header, err := ipv6.ParseHeader(data)
	if err != nil {
		return err
	}
	var dst inet256.Addr
	ip := netip.AddrFrom16(*(*[16]byte)(header.Dst))
	if e := p.getEntry(ip); e != nil {
		dst = e.addr
	} else {
		prefix, nbits, err := INET256PrefixFromIPv6(ip)
		if err != nil {
			return err
		}
		dst, err = node.FindAddr(ctx, prefix, nbits)
		if err != nil {
			return err
		}
		ent := &cacheEntry{addr: dst, createdAt: time.Now()}
		p.putEntry(ip, ent)
	}
	return node.Send(ctx, dst, data)
}

func (p *portal) inboundLoop(ctx context.Context) error {
	var msg inet256.Message
	for {
		if err := inet256.Receive(ctx, p.node, &msg); err != nil {
			return err
		}
		if !p.af(msg.Src) {
			logctx.Warnln(ctx, "inbound: ignoring INET256 message from: ", msg.Src)
			continue
		}
		if err := p.handleInbound(msg.Src, msg.Payload); err != nil {
			return err
		}
	}
}

func (p *portal) handleInbound(src inet256.Addr, data []byte) error {
	header, err := ipv6.ParseHeader(data)
	if err != nil {
		return err
	}
	srcIP := IPv6FromINET256(src)
	if !header.Src.Equal(srcIP.AsSlice()) {
		return errors.Errorf("dropping inbound message from wrong source")
	}
	_, err = p.dev.Write(data, 0)
	return err
}

func (p *portal) putEntry(key netip.Addr, e *cacheEntry) {
	p.cache.LoadOrStore(key, e)
}

func (p *portal) getEntry(key netip.Addr) *cacheEntry {
	v, ok := p.cache.Load(key)
	if !ok {
		return nil
	}
	return v.(*cacheEntry)
}

func configureInterface(ctx context.Context, iface string, ipAddr netip.Addr) error {
	switch runtime.GOOS {
	case "darwin":
		return ifconfigCmd(ctx, iface, "inet6", ipAddr.String(), "prefixlen", strconv.Itoa(netPrefix.Bits()))
	default:
		return errors.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func ifconfigCmd(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "ifconfig", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	logctx.Infoln(ctx, "ifconfig output:", string(output))
	return nil
}

type cacheEntry struct {
	addr      inet256.Addr
	createdAt time.Time
}
