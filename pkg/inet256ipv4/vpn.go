package inet256ipv4

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/ipv4"
	"golang.zx2c4.com/wireguard/tun"

	"github.com/inet256/inet256/pkg/inet256"
)

type VPN struct {
	inbound  map[inet256.Addr]IPv4
	outbound map[IPv4]inet256.Addr
	mtu      int
	localIP  IPv4

	tundev tun.Device
	node   *inet256.Node

	cf   context.CancelFunc
	done chan error
}

func NewVPN(n *inet256.Node, ifname string, conf *VPNConfig) (*VPN, error) {
	inbound := make(map[inet256.Addr]IPv4)
	outbound := make(map[IPv4]inet256.Addr)

	for _, hostConf := range conf.Hosts {
		inbound[hostConf.INet256] = hostConf.IP
		outbound[hostConf.IP] = hostConf.INet256
	}
	localIP := conf.LocalIP

	mtu := n.MinMTU()
	tundev, err := tun.CreateTUN(ifname, mtu)
	if err != nil {
		return nil, err
	}

	ctx, cf := context.WithCancel(context.Background())
	vpn := &VPN{
		inbound:  inbound,
		outbound: outbound,
		mtu:      mtu,
		localIP:  localIP,

		tundev: tundev,
		node:   n,

		done: make(chan error),
		cf:   cf,
	}
	go vpn.run(ctx)

	return vpn, nil
}

func (vpn *VPN) run(ctx context.Context) {
	defer close(vpn.done)

	vpn.node.OnRecv(vpn.handleRecv)
	defer vpn.node.OnRecv(inet256.NoOpRecvFunc)

	vpn.readLoop(ctx)
}

func (vpn *VPN) handleRecv(src, dst inet256.Addr, data []byte) {
	srcIP, exists := vpn.inbound[src]
	if !exists {
		return
	}
	if err := vpn.writeOne(srcIP, data); err != nil {
		log.Error(err)
	}
}

func (vpn *VPN) writeOne(srcIP IPv4, data []byte) error {
	log.Warn("incoming not implemented")
	return nil

	packet := []byte{}
	packet = append(packet, data...)
	if _, err := vpn.tundev.Write(packet, 0); err != nil {
		return err
	}
	return nil
}

func (vpn *VPN) readLoop(ctx context.Context) {
	buf := make([]byte, vpn.mtu)
	for {
		n, err := vpn.tundev.Read(buf, 0)
		if err != nil {
			log.Error(err)
			return
		}
		if err := vpn.readOne(ctx, buf[:n]); err != nil {
			log.Error(err)
		}
	}
}

func (vpn *VPN) readOne(ctx context.Context, buf []byte) error {
	if len(buf) < 1 {
		return errors.New("len(buffer) < 1")
	}

	version := buf[0] >> 4
	switch version {
	case ipv4.Version:
		srcIP := getSrc(buf)
		if srcIP != vpn.localIP {
			return nil
		}

		dstIP := getDst(buf)
		dstAddr, exists := vpn.outbound[dstIP]
		if !exists {
			return fmt.Errorf("no inet256 address for ip %v", dstIP)
		}
		payload, err := getPayload(buf)
		if err != nil {
			return err
		}
		return vpn.node.SendTo(ctx, dstAddr, payload)

	default:
		return fmt.Errorf("ip version %d not supported", version)
	}
}

func (vpn *VPN) Close() error {
	vpn.cf()
	if err := vpn.tundev.Close(); err != nil {
		return err
	}
	return <-vpn.done
}
