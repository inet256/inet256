package inet256ipv6

import (
	"context"

	"go.brendoncarroll.net/stdctx/logctx"
	"go.inet256.org/inet256/src/inet256"
)

type IPv6Addr = [16]byte

type NATTable struct {
	srv inet256.Service

	outbound map[IPv6Addr]inet256.Addr
	inbound  map[inet256.Addr]IPv6Addr
	vnodes   map[inet256.Addr]inet256.Node
}

func NewNATTable(srv inet256.Service) *NATTable {
	return &NATTable{
		srv:      srv,
		outbound: make(map[IPv6Addr]inet256.Addr),
		inbound:  make(map[inet256.Addr]IPv6Addr),
		vnodes:   make(map[inet256.Addr]inet256.Node),
	}
}

func (nt *NATTable) AddClient(ctx context.Context, ipv6 IPv6Addr) inet256.Addr {
	inside := ipv6
	_, priv, err := inet256.GenerateKey()
	if err != nil {
		panic(err)
	}
	vnode, err := nt.srv.Open(ctx, priv)
	if err != nil {
		panic(err)
	}
	outside := vnode.LocalAddr()

	nt.inbound[outside] = inside
	nt.outbound[inside] = outside
	nt.vnodes[outside] = vnode

	return outside
}

func (nt *NATTable) DeleteClient(ctx context.Context, ip6 IPv6Addr) {
	addr, exists := nt.outbound[ip6]
	if !exists {
		return
	}
	if err := nt.vnodes[addr].Close(); err != nil {
		logctx.Errorln(ctx, err)
	}
	delete(nt.outbound, ip6)
	delete(nt.inbound, addr)
	delete(nt.vnodes, addr)
}

func (nt *NATTable) NodeByInner(ipv6 IPv6Addr) inet256.Node {
	addr, exists := nt.outbound[ipv6]
	if !exists {
		return nil
	}
	return nt.vnodes[addr]
}

func (nt *NATTable) NodeByOuter(addr inet256.Addr) inet256.Node {
	return nt.vnodes[addr]
}
