package mesh256

import (
	"context"
	"sync"

	"github.com/brendoncarroll/go-p2p/f/x509"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/vswarm"
	"github.com/brendoncarroll/go-tai64"
	"github.com/brendoncarroll/stdctx"
	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/pkg/errors"
	"golang.org/x/exp/slog"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/peers"
)

const nameMemSwarm = "memory"

const DefaultQueueLen = 4

type Service interface {
	inet256.Service

	MainAddr(context.Context) (Addr, error)
	TransportAddrs(context.Context) ([]TransportAddr, error)
	PeerStatus(context.Context) ([]PeerStatus, error)
}

type PeerStatus struct {
	Addr       Addr
	LastSeen   map[string]tai64.TAI64N
	Uploaded   uint64
	Downloaded uint64
}

type Params NodeParams

type Server struct {
	params Params

	memrealm     *memswarm.SecureRealm[x509.PublicKey]
	mainID       inet256.Addr
	mainMemPeers peers.Store[multiswarm.Addr]
	mainMemSwarm *vswarm.SecureSwarm[memswarm.Addr, x509.PublicKey]
	mainNode     *node

	mu    sync.Mutex
	nodes map[inet256.Addr]*node
}

func NewServer(params Params) *Server {
	r := memswarm.NewSecureRealm[x509.PublicKey](memswarm.WithQueueLen(DefaultQueueLen))
	msw := r.NewSwarm(PublicKeyFromINET256(params.PrivateKey.Public()))
	if params.Swarms == nil {
		params.Swarms = make(map[string]multiswarm.DynSwarm, 1)
	}
	params.Swarms[nameMemSwarm] = multiswarm.WrapSecureSwarm[memswarm.Addr, x509.PublicKey](msw)

	memPeers := peers.NewStore[TransportAddr]()

	s := &Server{
		params: params,

		memrealm:     r,
		mainID:       inet256.NewAddr(params.PrivateKey.Public()),
		mainMemPeers: memPeers,
		mainMemSwarm: msw,
		mainNode: NewNode(NodeParams{
			Background: stdctx.Child(params.Background, "main"),
			PrivateKey: params.PrivateKey,
			Swarms:     params.Swarms,
			NewNetwork: params.NewNetwork,
			Peers:      peers.ChainStore[TransportAddr]{memPeers, params.Peers},
		}).(*node),
		nodes: make(map[inet256.Addr]*node),
	}
	return s
}

func (s *Server) Open(ctx context.Context, privateKey inet256.PrivateKey, opts ...inet256.NodeOption) (Node, error) {
	id := inet256.NewAddr(privateKey.Public())
	if id == s.mainID {
		return nil, errors.Errorf("clients cannot use main node's key")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.nodes[id]; exists {
		return nil, errors.New("node is already open")
	}
	swarm := s.memrealm.NewSwarm(PublicKeyFromINET256(privateKey.Public()))

	ps := peers.NewStore[TransportAddr]()
	ps.Add(s.mainID)
	ps.SetAddrs(s.mainID, []multiswarm.Addr{{Scheme: nameMemSwarm, Addr: s.mainMemSwarm.LocalAddrs()[0]}})
	s.mainMemPeers.Add(id)
	s.mainMemPeers.SetAddrs(id, []multiswarm.Addr{{Scheme: nameMemSwarm, Addr: swarm.LocalAddrs()[0]}})

	n := NewNode(NodeParams{
		Background: stdctx.Child(s.params.Background, id.Base64String()),
		NewNetwork: s.params.NewNetwork,
		Peers:      ps,
		PrivateKey: privateKey,
		Swarms: map[string]multiswarm.DynSwarm{
			nameMemSwarm: multiswarm.WrapSecureSwarm[memswarm.Addr, x509.PublicKey](swarm),
		},
	})
	s.nodes[id] = n.(*node)
	logctx.Info(ctx, "created node", slog.Any("addr", id))
	return n, nil
}

func (s *Server) Drop(ctx context.Context, privateKey inet256.PrivateKey) error {
	id := inet256.NewAddr(privateKey.Public())
	s.mu.Lock()
	defer s.mu.Unlock()
	n, exists := s.nodes[id]
	if !exists {
		return nil
	}
	err := n.Close()
	delete(s.nodes, id)
	return err
}

func (s *Server) LookupPublicKey(ctx context.Context, target Addr) (inet256.PublicKey, error) {
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

func (s *Server) TransportAddrs(ctx context.Context) ([]multiswarm.Addr, error) {
	return s.mainNode.TransportAddrs(), nil
}

func (s *Server) PeerStatus(ctx context.Context) ([]PeerStatus, error) {
	var ret []PeerStatus
	mainNode := s.mainNode
	for _, id := range s.params.Peers.ListPeers() {
		lastSeen := mainNode.LastSeen(id)
		ret = append(ret, PeerStatus{
			Addr:       id,
			LastSeen:   lastSeen,
			Uploaded:   mainNode.mhSwarm.GetTx(id),
			Downloaded: mainNode.mhSwarm.GetRx(id),
		})
	}
	return ret, nil
}

func (s *Server) MainAddr(ctx context.Context) (Addr, error) {
	return s.MainNode().LocalAddr(), nil
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
	s.nodes = make(map[inet256.Addr]*node)
	s.mainNode.Close()
	return nil
}
