package inet256grpc

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"io"
	mrand "math/rand"
	sync "sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ INET256Server = &Server{}

type Server struct {
	s *inet256.Server

	mu     sync.RWMutex
	nodes  map[p2p.PeerID]inet256.Node
	active map[p2p.PeerID][]INET256_ConnectServer

	UnimplementedINET256Server
}

func NewServer(s *inet256.Server) *Server {
	return &Server{
		s:      s,
		nodes:  make(map[p2p.PeerID]inet256.Node),
		active: make(map[p2p.PeerID][]INET256_ConnectServer),
	}
}

func (s *Server) GenerateKey(ctx context.Context, _ *empty.Empty) (*GenerateKeyRes, error) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	keyData, err := inet256.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	return &GenerateKeyRes{
		PrivateKey: keyData,
	}, nil
}

func (s *Server) LookupSelf(ctx context.Context, req *LookupSelfReq) (*PeerInfo, error) {
	privKey, err := inet256.ParsePrivateKey(req.PrivateKey)
	if err != nil {
		return nil, err
	}
	privKey2, ok := privKey.(interface{ Public() crypto.PublicKey })
	if !ok {
		return nil, errors.Errorf("unsupported key")
	}
	pubKey := privKey2.Public()
	peerID := p2p.NewPeerID(pubKey)
	return &PeerInfo{
		Addr:      peerID[:],
		PublicKey: p2p.MarshalPublicKey(pubKey),
	}, nil
}

func (s *Server) Lookup(ctx context.Context, req *LookupReq) (*PeerInfo, error) {
	target := inet256.Addr{}
	copy(target[:], req.TargetAddr)
	pubKey, err := s.s.LookupPublicKey(ctx, target)
	if err != nil {
		return nil, err
	}
	return &PeerInfo{
		Addr:      target[:],
		PublicKey: p2p.MarshalPublicKey(pubKey),
	}, nil
}

func (s *Server) MTU(ctx context.Context, req *MTUReq) (*MTURes, error) {
	target := inet256.AddrFromBytes(req.Target)
	mtu := s.s.MTU(ctx, target)
	return &MTURes{
		Mtu: int64(mtu),
	}, nil
}

func (s *Server) Connect(srv INET256_ConnectServer) error {
	msg, err := srv.Recv()
	if err != nil {
		return err
	}
	if msg.ConnectInit == nil {
		return errors.Errorf("first message must contain ConnectInit")
	}
	cinit := msg.ConnectInit
	privKey, err := inet256.ParsePrivateKey(cinit.PrivateKey)
	if err != nil {
		return err
	}
	id := p2p.NewPeerID(privKey.Public())
	if err := s.addServer(privKey, id, srv); err != nil {
		return err
	}
	defer s.removeServer(id, srv)

	ctx := srv.Context()
	for {
		msg, err := srv.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if msg.Datagram == nil {
			continue
		}
		dst := inet256.Addr{}
		if len(msg.Datagram.Dst) < 32 {
			if dst, err = s.findAddr(ctx, msg.Datagram.Dst); err != nil {
				return err
			}
		} else {
			copy(dst[:], msg.Datagram.Dst)
		}
		if err := s.fromClient(ctx, id, dst, msg.Datagram.Payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) findAddr(ctx context.Context, prefix []byte) (inet256.Addr, error) {
	return s.s.FindAddr(ctx, prefix, len(prefix)*8)
}

func (s *Server) toClients(src, dst inet256.Addr, payload []byte) {
	s.mu.RLock()
	var srv INET256_ConnectServer
	srvs := s.active[dst]
	if len(srvs) > 0 {
		srv = srvs[mrand.Intn(len(srvs))]
	}
	s.mu.RUnlock()

	if srv == nil {
		logrus.Error("no active clients")
		return
	}

	if err := srv.Send(&ConnectMsg{
		Datagram: &Datagram{
			Src:     src[:],
			Dst:     dst[:],
			Payload: payload,
		},
	}); err != nil {
		logrus.Error(err)
	}
}

func (s *Server) fromClient(ctx context.Context, src, dst inet256.Addr, payload []byte) error {
	s.mu.RLock()
	n := s.nodes[src]
	s.mu.RUnlock()
	return n.Tell(ctx, dst, payload)
}

func (s *Server) addServer(privKey p2p.PrivateKey, id p2p.PeerID, srv INET256_ConnectServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.nodes[id]
	if !exists {
		n, err := s.s.CreateNode(privKey)
		if err != nil {
			return err
		}
		s.nodes[id] = n
		n.OnRecv(s.toClients)
	}
	s.active[id] = append(s.active[id], srv)
	return nil
}

func (s *Server) removeServer(id p2p.PeerID, srv INET256_ConnectServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	srvs := s.active[id]
	for i := range srvs {
		if srv == srvs[i] {
			if i < len(srvs)-1 {
				srvs = append(srvs[:i], srvs[i+1:]...)
			} else {
				srvs = srvs[:i]
			}
			return
		}
	}
	s.active[id] = srvs

	if len(s.active[id]) > 0 {
		return
	}
	if err := s.nodes[id].Close(); err != nil {
		logrus.Error(err)
	}
	delete(s.nodes, id)
}
