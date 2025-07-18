package inet256mem

import (
	"context"
	"errors"
	"sync"

	"go.brendoncarroll.net/p2p"
	"go.brendoncarroll.net/stdctx/logctx"

	"go.inet256.org/inet256/src/inet256"
	"go.inet256.org/inet256/src/internal/netutil"
)

const defaultQueueLen = 4

type memService struct {
	config config

	mu    sync.RWMutex
	nodes map[inet256.Addr]*memNode
}

func New(opts ...Option) inet256.Service {
	config := config{
		queueLen: defaultQueueLen,
	}
	for _, opt := range opts {
		opt(&config)
	}
	return &memService{
		config: config,
		nodes:  make(map[inet256.Addr]*memNode),
	}
}

func (s *memService) Open(ctx context.Context, privKey inet256.PrivateKey, opts ...inet256.NodeOption) (inet256.Node, error) {
	id := inet256.NewID(privKey.Public().(inet256.PublicKey))
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
	id := inet256.NewID(privKey.Public().(inet256.PublicKey))
	s.delete(id)
	return nil
}

func (s *memService) delete(id inet256.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if node, exists := s.nodes[id]; exists {
		node.incoming.Close()
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
	incoming  netutil.Queue
}

func newMemNode(s *memService, privKey inet256.PrivateKey) *memNode {
	return &memNode{
		s:         s,
		publicKey: privKey.Public().(inet256.PublicKey),
		incoming:  netutil.NewQueue(s.config.queueLen),
	}
}

func (node *memNode) Send(ctx context.Context, dst inet256.Addr, data []byte) error {
	dstNode := node.s.getNode(dst)
	if dstNode != nil {
		accepted := dstNode.incoming.Deliver(p2p.Message[inet256.Addr]{
			Src:     node.LocalAddr(),
			Dst:     dst,
			Payload: data,
		})
		if !accepted {
			logctx.Warnf(ctx, "inet256mem: dropped message len=%d dst=%v", len(data), dst)
		}
	}
	return nil
}

func (node *memNode) Receive(ctx context.Context, fn inet256.ReceiveFunc) error {
	return node.incoming.Receive(ctx, func(x p2p.Message[inet256.Addr]) {
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
	return inet256.NewID(node.publicKey)
}

func (node *memNode) PublicKey() inet256.PublicKey {
	return node.publicKey
}

func (node *memNode) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	return node.s.findAddr(prefix, nbits)
}

func (node *memNode) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	node.s.mu.RLock()
	defer node.s.mu.RUnlock()
	if node, exists := node.s.nodes[target]; exists {
		return node.publicKey, nil
	}
	return nil, inet256.ErrPublicKeyNotFound
}

func (node *memNode) MTU() int {
	return inet256.MTU
}
