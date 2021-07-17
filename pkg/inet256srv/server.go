package inet256srv

import (
	"context"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const nameMemSwarm = "memory"

type Service interface {
	inet256.Service

	MainAddr() Addr
	TransportAddrs() []p2p.Addr
	PeerStatus() []PeerStatus
}

type PeerStatus struct {
	Addr       Addr
	LastSeen   map[string]time.Time
	Uploaded   uint64
	Downloaded uint64
}

type Server struct {
	params Params

	memrealm     *memswarm.Realm
	mainID       inet256.Addr
	mainMemPeers PeerStore
	mainMemSwarm *memswarm.Swarm
	mainNode     Node

	mu    sync.Mutex
	nodes map[inet256.Addr]Node

	log *logrus.Logger
}

func NewServer(params Params) *Server {
	swarms := make(map[string]p2p.SecureSwarm)
	for k, v := range params.Swarms {
		swarms[k] = v
	}

	r := memswarm.NewRealm()
	msw := r.NewSwarmWithKey(params.PrivateKey)
	swarms[nameMemSwarm] = msw

	memPeers := NewPeerStore()

	s := &Server{
		params: params,

		memrealm:     r,
		mainID:       inet256.NewAddr(params.PrivateKey.Public()),
		mainMemPeers: memPeers,
		mainMemSwarm: msw,
		mainNode: NewNode(Params{
			PrivateKey: params.PrivateKey,
			Swarms:     swarms,
			Networks:   params.Networks,
			Peers:      ChainPeerStore{memPeers, params.Peers},
		}),
		nodes: make(map[inet256.Addr]Node),
		log:   logrus.New(),
	}
	return s
}

func (s *Server) CreateNode(ctx context.Context, privateKey p2p.PrivateKey) (Node, error) {
	id := inet256.NewAddr(privateKey.Public())
	n, err := func() (Node, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, exists := s.nodes[id]; exists {
			return nil, errors.Errorf("node already exists")
		}
		swarm := s.memrealm.NewSwarmWithKey(privateKey)

		ps := NewPeerStore()
		ps.Add(s.mainID)
		ps.SetAddrs(s.mainID, []p2p.Addr{multiswarm.Addr{Transport: nameMemSwarm, Addr: s.mainMemSwarm.LocalAddrs()[0]}})
		s.mainMemPeers.Add(id)
		s.mainMemPeers.SetAddrs(id, []p2p.Addr{multiswarm.Addr{Transport: nameMemSwarm, Addr: swarm.LocalAddrs()[0]}})

		n := NewNode(Params{
			Networks:   s.params.Networks,
			Peers:      ps,
			PrivateKey: privateKey,
			Swarms: map[string]p2p.SecureSwarm{
				nameMemSwarm: swarm,
			},
		})
		s.nodes[id] = n
		return n, nil
	}()
	if err != nil {
		return nil, err
	}
	if err := n.Bootstrap(ctx); err != nil {
		return nil, errors.Wrapf(err, "while bootstrapping node")
	}
	s.log.WithFields(logrus.Fields{"addr": id}).Info("created node")
	return n, nil
}

func (s *Server) DeleteNode(privateKey p2p.PrivateKey) error {
	id := inet256.NewAddr(privateKey.Public())
	s.mu.Lock()
	defer s.mu.Unlock()
	n, exists := s.nodes[id]
	if !exists {
		return nil
	}
	err := n.Close()
	s.mainMemPeers.Remove(id)
	delete(s.nodes, id)
	s.log.WithFields(logrus.Fields{"addr": id}).Info("deleted node")
	return err
}

func (s *Server) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	return s.mainNode.LookupPublicKey(ctx, target)
}

func (s *Server) MTU(ctx context.Context, target Addr) int {
	return s.mainNode.MTU(ctx, target)
}

func (s *Server) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	return s.mainNode.FindAddr(ctx, prefix, nbits)
}

func (s *Server) LocalAddr() Addr {
	return s.mainNode.LocalAddr()
}

func (s *Server) TransportAddrs() (ret []p2p.Addr) {
	return s.mainNode.(*node).TransportAddrs()
}

func (s *Server) PeerStatus() []PeerStatus {
	var ret []PeerStatus
	mainNode := s.mainNode.(*node)
	for _, id := range s.params.Peers.ListPeers() {
		lastSeen := mainNode.LastSeen(id)
		ret = append(ret, PeerStatus{
			Addr:       id,
			LastSeen:   lastSeen,
			Uploaded:   mainNode.basePeerSwarm.GetTx(id),
			Downloaded: mainNode.basePeerSwarm.GetRx(id),
		})
	}
	return ret
}

func (s *Server) MainAddr() Addr {
	return s.MainNode().LocalAddr()
}

func (s *Server) MainNode() Node {
	return s.mainNode
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range s.nodes {
		n.Close()
	}
	return s.mainNode.Close()
}

func marshalAddr(a p2p.Addr) string {
	data, err := a.MarshalText()
	if err != nil {
		panic(err)
	}
	return string(data)
}
