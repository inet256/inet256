package inet256ipv6

import (
	"crypto/ed25519"

	"github.com/inet256/inet256/client/go_client/inet256client"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256grpc"
	log "github.com/sirupsen/logrus"
)

type IPv6Addr = [16]byte

type NATTable struct {
	client inet256grpc.INET256Client

	outbound map[IPv6Addr]inet256.Addr
	inbound  map[inet256.Addr]IPv6Addr
	vnodes   map[inet256.Addr]inet256.Node
}

func NewNATTable(client inet256grpc.INET256Client) *NATTable {
	return &NATTable{
		client:   client,
		outbound: make(map[IPv6Addr]inet256.Addr),
		inbound:  make(map[inet256.Addr]IPv6Addr),
		vnodes:   make(map[inet256.Addr]inet256.Node),
	}
}

func (nt *NATTable) AddClient(ipv6 IPv6Addr) inet256.Addr {
	inside := ipv6
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	vnode, err := inet256client.NewFromGRPC(nt.client, priv)
	if err != nil {
		panic(err)
	}
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
