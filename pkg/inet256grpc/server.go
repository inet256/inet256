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
	UnimplementedAdminServer
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
	keyData := serde.MarshalPrivateKey(priv)
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
	mainAddr, err := srv.MainAddr()
	if err != nil {
		return nil, err
	}
	stati, err := srv.PeerStatus()
	if err != nil {
		return nil, err
	}
	taddrs, err := srv.TransportAddrs()
	if err != nil {
		return nil, err
	}
	return &Status{
		LocalAddr:      mainAddr[:],
		PeerStatus:     PeerStatusToProto(stati),
		TransportAddrs: serde.MarshalAddrs(taddrs),
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
	node, err := s.s.CreateNode(ctx, privKey)
	if err != nil {
		return err
	}
	defer func() {
		if err := node.Close(); err != nil {
			logrus.Errorf("error closing node: %v", err)
		}
	}()
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
		var msg inet256.Message
		for {
			if err := inet256.Receive(ctx, node, &msg); err != nil {
				return err
			}
			if err := srv.Send(&ConnectMsg{
				Datagram: &Datagram{
					Src:     msg.Src[:],
					Dst:     msg.Dst[:],
					Payload: msg.Payload,
				},
			}); err != nil {
				return err
			}
		}
	})
	return eg.Wait()
}
