package rcsrv

import (
	"context"
	"sync"

	"go.inet256.org/inet256/pkg/inet256"
)

type Server struct {
	Service inet256.Service

	mu    sync.Mutex
	nodes map[inet256.Addr]*nodeState
}

func Wrap(x inet256.Service) *Server {
	return &Server{
		Service: x,
		nodes:   make(map[inet256.Addr]*nodeState),
	}
}

func (s *Server) Open(ctx context.Context, pk inet256.PrivateKey, opts ...inet256.NodeOption) (inet256.Node, error) {
	id := inet256.NewAddr(pk.Public())
	s.mu.Lock()
	defer s.mu.Unlock()
	ns, exists := s.nodes[id]
	if exists {
		ns.count++
		return rcNode{s: s, Node: ns.node}, nil
	}
	node, err := s.Service.Open(ctx, pk, opts...)
	if err != nil {
		return nil, err
	}
	s.nodes[id] = &nodeState{node: node, count: 1}
	return rcNode{s: s, Node: node}, nil
}

func (s *Server) Drop(ctx context.Context, pk inet256.PrivateKey) error {
	id := inet256.NewAddr(pk.Public())
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.delete(id)
}

func (s *Server) delete(id inet256.ID) (err error) {
	ns, exists := s.nodes[id]
	if exists {
		err = ns.node.Close()
	}
	delete(s.nodes, id)
	return err
}

type nodeState struct {
	node  inet256.Node
	count int
}

type rcNode struct {
	s *Server
	inet256.Node
}

func (n rcNode) Close() error {
	id := n.Node.LocalAddr()
	n.s.mu.Lock()
	defer n.s.mu.Unlock()
	ns, exists := n.s.nodes[id]
	if !exists {
		return nil
	}
	ns.count--
	if ns.count < 1 {
		return n.s.delete(id)
	}
	return nil
}
