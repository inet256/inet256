package inet256grpc

import (
	"context"
	"crypto/ed25519"
	"io"
	sync "sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ INET256Server = &Server{}

type Server struct {
	s inet256.Service

	mu     sync.Mutex
	nodes  map[inet256.Addr]inet256.Node
	counts map[inet256.Addr]int

	UnimplementedINET256Server
	UnimplementedManagementServer
}

func NewServer(s inet256.Service) *Server {
	return &Server{
		s:      s,
		nodes:  make(map[inet256.Addr]inet256.Node),
		counts: make(map[inet256.Addr]int),
	}
}

func (s *Server) GenerateKey(ctx context.Context, _ *empty.Empty) (*GenerateKeyRes, error) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	keyData, err := serde.MarshalPrivateKey(priv)
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
	srv, ok := s.s.(*inet256srv.Server)
	if !ok {
		return nil, errors.Errorf("server does not support GetStatus")
	}
	mainAddr := srv.MainAddr()
	return &Status{
		LocalAddr:      mainAddr[:],
		PeerStatus:     PeerStatusToProto(srv.PeerStatus()),
		TransportAddrs: serde.MarshalAddrs(srv.TransportAddrs()),
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
	privKey, err := serde.ParsePrivateKey(cinit.PrivateKey)
	if err != nil {
		return err
	}
	node, err := s.getOrCreateNode(ctx, privKey)
	if err != nil {
		return err
	}
	defer s.decrNode(privKey)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		for {
			msg, err := srv.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if msg.ConnectInit != nil {
				return errors.Errorf("cannot send ConnectInit after first message")
			}
			if msg.Datagram == nil {
				continue
			}
			dst := inet256.Addr{}
			copy(dst[:], msg.Datagram.Dst)
			if err := func() error {
				ctx, cf := context.WithTimeout(context.Background(), 3*time.Second)
				defer cf()
				return node.Tell(ctx, dst, msg.Datagram.Payload)
			}(); err != nil {
				return err
			}
		}
	})
	eg.Go(func() error {
		buf := make([]byte, inet256.MaxMTU)
		for {
			var src, dst inet256.Addr
			n, err := node.Receive(ctx, &src, &dst, buf)
			if err != nil {
				return err
			}
			if err := srv.Send(&ConnectMsg{
				Datagram: &Datagram{
					Src:     src[:],
					Dst:     dst[:],
					Payload: buf[:n],
				},
			}); err != nil {
				return err
			}
		}
	})
	return eg.Wait()
}

func (s *Server) getOrCreateNode(ctx context.Context, privKey p2p.PrivateKey) (inet256.Node, error) {
	id := inet256.NewAddr(privKey.Public())
	s.mu.Lock()
	defer s.mu.Unlock()
	node, exists := s.nodes[id]
	if !exists {
		ctx, cf := context.WithTimeout(ctx, 30*time.Second)
		defer cf()
		var err error
		node, err = s.s.CreateNode(ctx, privKey)
		if err != nil {
			return nil, err
		}
		s.nodes[id] = node
	}
	s.counts[id]++
	count := s.counts[id]
	logrus.WithFields(logrus.Fields{"addr": id, "count": count}).Info("starting gRPC stream")
	return node, nil
}

func (s *Server) decrNode(privKey p2p.PrivateKey) {
	id := inet256.NewAddr(privKey.Public())
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.nodes[id]
	if !exists {
		panic("decr non-existing node")
	}
	s.counts[id]--
	count := s.counts[id]
	logrus.WithFields(logrus.Fields{"addr": id, "count": count}).Info("closing gRPC stream")
	if s.counts[id] == 0 {
		if err := s.s.DeleteNode(privKey); err != nil {
			logrus.Error("while closing node: ", err)
		}
		delete(s.counts, id)
		delete(s.nodes, id)
	}
}
