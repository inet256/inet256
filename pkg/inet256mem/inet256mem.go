package inet256mem

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
)

type memService struct {
	mu    sync.RWMutex
	nodes map[inet256.Addr]*memNode
}

func New() inet256.Service {
	return &memService{
		nodes: make(map[inet256.Addr]*memNode),
	}
}

func (s *memService) Open(ctx context.Context, privKey inet256.PrivateKey, opts ...inet256.NodeOption) (inet256.Node, error) {
	id := inet256.NewAddr(privKey.Public())
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.nodes[id]; exists {
		return nil, errors.New("node already open")
	}
	node := newMemNode(s, privKey)
	s.nodes[id] = node
	return node, nil
}

func (s *memService) Drop(ctx context.Context, privKey inet256.PrivateKey) error {
	id := inet256.NewAddr(privKey.Public())
	s.delete(id)
	return nil
}

func (s *memService) delete(id inet256.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if node, exists := s.nodes[id]; exists {
		node.hub.CloseWithError(net.ErrClosed)
		delete(s.nodes, id)
	}
}

func (s *memService) findAddr(prefix []byte, nbits int) (inet256.Addr, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id := range s.nodes {
		if inet256.HasPrefix(id[:], prefix, nbits) {
			return id, nil
		}
	}
	return inet256.Addr{}, inet256.ErrNoAddrWithPrefix
}

func (s *memService) getNode(x inet256.Addr) *memNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nodes[x]
}

type memNode struct {
	s         *memService
	publicKey inet256.PublicKey
	hub       *netutil.TellHub
}

func newMemNode(s *memService, privKey inet256.PrivateKey) *memNode {
	return &memNode{
		s:         s,
		publicKey: privKey.Public(),
		hub:       netutil.NewTellHub(),
	}
}

func (node *memNode) Send(ctx context.Context, dst inet256.Addr, data []byte) error {
	dstNode := node.s.getNode(dst)
	if dstNode != nil {
		return dstNode.hub.Deliver(ctx, p2p.Message[inet256.Addr]{
			Src:     node.LocalAddr(),
			Dst:     dst,
			Payload: data,
		})
	}
	return nil
}

func (node *memNode) Receive(ctx context.Context, fn inet256.ReceiveFunc) error {
	return node.hub.Receive(ctx, func(x p2p.Message[inet256.Addr]) {
		fn(inet256.Message{
			Src:     x.Src,
			Dst:     x.Dst,
			Payload: x.Payload,
		})
	})
}

func (node *memNode) Close() error {
	node.s.delete(node.LocalAddr())
	return nil
}

func (node *memNode) LocalAddr() inet256.Addr {
	return inet256.NewAddr(node.publicKey)
}

func (node *memNode) PublicKey() inet256.PublicKey {
	return node.publicKey
}

func (node *memNode) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	return node.s.findAddr(prefix, nbits)
}

func (node *memNode) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	node.s.mu.RLock()
	defer node.s.mu.RLock()
	if node, exists := node.s.nodes[target]; exists {
		return node.publicKey, nil
	}
	return nil, inet256.ErrPublicKeyNotFound
}

func (node *memNode) MTU(ctx context.Context, target inet256.Addr) int {
	return inet256.MaxMTU
}
