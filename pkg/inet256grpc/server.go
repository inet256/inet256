package inet256grpc

import (
	"context"
	"crypto/ed25519"
	"io"
	mrand "math/rand"
	sync "sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ INET256Server = &Server{}

type Server struct {
	s inet256.Service

	mu     sync.RWMutex
	nodes  map[p2p.PeerID]inet256.Node
	active map[p2p.PeerID][]INET256_ConnectServer

	UnimplementedINET256Server
}

func NewServer(s inet256.Service) *Server {
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

func (s *Server) Lookup(ctx context.Context, req *LookupReq) (*LookupRes, error) {
	target := inet256.Addr{}
	copy(target[:], req.TargetAddr)
	pubKey, err := s.s.LookupPublicKey(ctx, target)
	if err != nil {
		return nil, err
	}
	return &LookupRes{
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

func (s *Server) GetStatus(ctx context.Context, _ *emptypb.Empty) (*Status, error) {
	mainAddr := s.s.MainAddr()
	return &Status{
		LocalAddr:      mainAddr[:],
		PeerStatus:     PeerStatusToProto(s.s.PeerStatus()),
		TransportAddrs: s.s.TransportAddrs(),
	}, nil
}

func (s *Server) Connect(srv INET256_ConnectServer) error {
	ctx := srv.Context()
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
	if err := s.addServer(ctx, privKey, id, srv); err != nil {
		return err
	}
	defer s.removeServer(id, privKey, srv)

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
		copy(dst[:], msg.Datagram.Dst)
		if err := s.fromClient(ctx, id, dst, msg.Datagram.Payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) toClientLoop(ctx context.Context, node inet256.Node) error {
	buf := make([]byte, inet256.TransportMTU)
	for {
		var src, dst inet256.Addr
		n, err := node.Recv(ctx, &src, &dst, buf)
		if err != nil {
			return err
		}
		if err := s.toClient(src, dst, buf[:n]); err != nil {
			logrus.Errorf("error sending to client: %v", err)
		}
	}
}

func (s *Server) toClient(src, dst inet256.Addr, payload []byte) error {
	s.mu.RLock()
	var srv INET256_ConnectServer
	srvs := s.active[dst]
	if len(srvs) > 0 {
		srv = srvs[mrand.Intn(len(srvs))]
	}
	s.mu.RUnlock()
	if srv == nil {
		return errors.Errorf("no active clients")
	}
	return srv.Send(&ConnectMsg{
		Datagram: &Datagram{
			Src:     src[:],
			Dst:     dst[:],
			Payload: payload,
		},
	})
}

func (s *Server) fromClient(ctx context.Context, src, dst inet256.Addr, payload []byte) error {
	s.mu.RLock()
	n := s.nodes[src]
	s.mu.RUnlock()
	return n.Tell(ctx, dst, payload)
}

func (s *Server) addServer(ctx context.Context, privKey p2p.PrivateKey, id p2p.PeerID, srv INET256_ConnectServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.nodes[id]
	if !exists {
		node, err := s.s.CreateNode(ctx, privKey)
		if err != nil {
			return err
		}
		s.nodes[id] = node
		go func() {
			if err := s.toClientLoop(ctx, node); err != nil {
				logrus.Error(err)
			}
		}()
	}
	s.active[id] = append(s.active[id], &serverWrapper{INET256_ConnectServer: srv})
	return nil
}

func (s *Server) removeServer(id p2p.PeerID, privKey p2p.PrivateKey, srv INET256_ConnectServer) {
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
			break
		}
	}
	s.active[id] = srvs

	if len(s.active[id]) > 0 {
		return
	}
	if err := s.s.DeleteNode(privKey); err != nil {
		logrus.Error(err)
	}
	delete(s.nodes, id)
}

type serverWrapper struct {
	INET256_ConnectServer
	mu sync.Mutex
}

func (s *serverWrapper) Send(m *ConnectMsg) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.INET256_ConnectServer.Send(m)
}
