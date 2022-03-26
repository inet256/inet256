package inet256srv

import (
	"context"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/peers"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const nameMemSwarm = "memory"

type Service interface {
	inet256.Service

	MainAddr() (Addr, error)
	TransportAddrs() ([]p2p.Addr, error)
	PeerStatus() ([]PeerStatus, error)
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
	mainMemPeers peers.Store[multiswarm.Addr]
	mainMemSwarm *memswarm.Swarm
	mainNode     Node

	mu    sync.Mutex
	nodes map[inet256.Addr]*rcNode
	rcs   map[inet256.Addr]int

	log *logrus.Logger
}

func NewServer(params Params) *Server {
	memLogger := logrus.StandardLogger()
	r := memswarm.NewRealm(memswarm.WithLogger(memLogger))
	msw := r.NewSwarmWithKey(params.PrivateKey)
	if params.Swarms == nil {
		params.Swarms = make(map[string]multiswarm.DynSwarm, 1)
	}
	params.Swarms[nameMemSwarm] = multiswarm.WrapSecureSwarm[memswarm.Addr](msw)

	memPeers := peers.NewStore[TransportAddr]()

	s := &Server{
		params: params,

		memrealm:     r,
		mainID:       inet256.NewAddr(params.PrivateKey.Public()),
		mainMemPeers: memPeers,
		mainMemSwarm: msw,
		mainNode: NewNode(Params{
			PrivateKey: params.PrivateKey,
			Swarms:     params.Swarms,
			NewNetwork: params.NewNetwork,
			Peers:      peers.ChainStore[TransportAddr]{memPeers, params.Peers},
		}),
		nodes: make(map[inet256.Addr]*rcNode),
		rcs:   make(map[inet256.Addr]int),
		log:   logrus.StandardLogger(),
	}
	return s
}

func (s *Server) Open(ctx context.Context, privateKey p2p.PrivateKey) (Node, error) {
	id := inet256.NewAddr(privateKey.Public())
	if id == s.mainID {
		return nil, errors.Errorf("clients cannot use main node's key")
	}
	node, err := func() (Node, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if node, exists := s.nodes[id]; exists {
			s.rcs[id]++
			s.log.WithFields(logrus.Fields{"addr": id, "count": s.rcs[id]}).Infof("opened ref to node")
			return node, nil
		}
		swarm := s.memrealm.NewSwarmWithKey(privateKey)

		ps := peers.NewStore[TransportAddr]()
		ps.Add(s.mainID)
		ps.SetAddrs(s.mainID, []multiswarm.Addr{{Transport: nameMemSwarm, Addr: s.mainMemSwarm.LocalAddrs()[0]}})
		s.mainMemPeers.Add(id)
		s.mainMemPeers.SetAddrs(id, []multiswarm.Addr{{Transport: nameMemSwarm, Addr: swarm.LocalAddrs()[0]}})

		n := NewNode(Params{
			NewNetwork: s.params.NewNetwork,
			Peers:      ps,
			PrivateKey: privateKey,
			Swarms: map[string]multiswarm.DynSwarm{
				nameMemSwarm: multiswarm.WrapSecureSwarm[memswarm.Addr](swarm),
			},
		})
		rcn := &rcNode{node: n.(*node), s: s}
		s.nodes[id] = rcn
		s.rcs[id] = 1
		s.log.WithFields(logrus.Fields{"addr": id, "count": s.rcs[id]}).Infof("created node")
		return rcn, nil
	}()
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (s *Server) Delete(ctx context.Context, privateKey p2p.PrivateKey) error {
	id := inet256.NewAddr(privateKey.Public())
	s.mu.Lock()
	defer s.mu.Unlock()
	rcn, exists := s.nodes[id]
	if !exists {
		return nil
	}
	err := rcn.node.Close()
	delete(s.rcs, id)
	delete(s.nodes, id)
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

func (s *Server) TransportAddrs() ([]multiswarm.Addr, error) {
	return s.mainNode.(*node).TransportAddrs(), nil
}

func (s *Server) PeerStatus() ([]PeerStatus, error) {
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
	return ret, nil
}

func (s *Server) MainAddr() (Addr, error) {
	return s.MainNode().LocalAddr(), nil
}

func (s *Server) MainNode() Node {
	return s.mainNode
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, n := range s.nodes {
		n.node.Close()
	}
	s.mainNode.Close()
	s.rcs = make(map[inet256.Addr]int)
	s.nodes = make(map[inet256.Addr]*rcNode)
	return nil
}

type rcNode struct {
	*node
	s *Server
}

func (rcn *rcNode) Close() error {
	s := rcn.s
	id := rcn.node.LocalAddr()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rcs[id]--
	log := s.log.WithFields(logrus.Fields{"addr": id, "count": s.rcs[id]})
	if s.rcs[id] < 1 {
		if err := rcn.node.Close(); err != nil {
			log.Warnf("error closing %v", err)
		}
		delete(s.nodes, id)
		delete(s.rcs, id)
		s.mainMemPeers.Remove(id)
		log.Infof("deleted node")
	} else {
		log.Info("dropped ref to node")
	}
	return nil
}
