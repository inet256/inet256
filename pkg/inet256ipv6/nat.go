package inet256ipv6

import (
	"github.com/inet256/inet256/pkg/inet256"
	log "github.com/sirupsen/logrus"
)

type IPv6Addr = [16]byte

type NATTable struct {
	node *inet256.Node

	outbound map[IPv6Addr]inet256.Addr
	inbound  map[inet256.Addr]IPv6Addr
	vnodes   map[inet256.Addr]*inet256.Node
}

func NewNATTable(node *inet256.Node) *NATTable {
	return &NATTable{
		node:     node,
		outbound: make(map[IPv6Addr]inet256.Addr),
		inbound:  make(map[inet256.Addr]IPv6Addr),
		vnodes:   make(map[inet256.Addr]*inet256.Node),
	}
}

func (nt *NATTable) AddClient(ipv6 IPv6Addr) inet256.Addr {
	inside := ipv6

	vnode := nt.node.NewVirtual()
	outside := vnode.LocalAddr()

	nt.inbound[outside] = inside
	nt.outbound[inside] = outside
	nt.vnodes[outside] = vnode

	return outside
}

func (nt *NATTable) DeleteClient(ip6 IPv6Addr) {
	addr, exists := nt.outbound[ip6]
	if !exists {
		return
	}
	if err := nt.vnodes[addr].Close(); err != nil {
		log.Error(err)
	}
	delete(nt.outbound, ip6)
	delete(nt.inbound, addr)
	delete(nt.vnodes, addr)
}

func (nt *NATTable) NodeByInner(ipv6 IPv6Addr) *inet256.Node {
	addr, exists := nt.outbound[ipv6]
	if !exists {
		return nil
	}
	return nt.vnodes[addr]
}

func (nt *NATTable) NodeByOuter(addr inet256.Addr) *inet256.Node {
	return nt.vnodes[addr]
}
