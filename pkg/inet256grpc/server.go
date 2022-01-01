package inet256grpc

import (
	"context"
	"crypto/ed25519"
	"io"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ INET256Server = &Server{}

type Server struct {
	s inet256.Service

	UnimplementedINET256Server
	UnimplementedAdminServer
}

func NewServer(s inet256srv.Service) *Server {
	return &Server{
		s: s,
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

func (s *Server) FindAddr(ctx context.Context, req *FindAddrReq) (*FindAddrRes, error) {
	addr, err := s.s.FindAddr(ctx, req.Prefix, int(req.Nbits))
	if err != nil {
		if errors.Is(err, inet256.ErrNoAddrWithPrefix) {
			err = status.Errorf(codes.NotFound, "%v", err)
		}
		return nil, err
	}
	return &FindAddrRes{
		Addr: addr[:],
	}, nil
}

func (s *Server) LookupPublicKey(ctx context.Context, req *LookupPublicKeyReq) (*LookupPublicKeyRes, error) {
	target := inet256.AddrFromBytes(req.Target)
	pubKey, err := s.s.LookupPublicKey(ctx, target)
	if err != nil {
		return nil, err
	}
	return &LookupPublicKeyRes{
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
		PeerStatus:     peerStatusToProto(stati),
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
	node, err := s.s.Open(ctx, privKey)
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
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			if msg.ConnectInit != nil {
				return errors.Errorf("cannot send ConnectInit after first message")
			}
			if msg.Datagram == nil {
				continue
			}
			dst := inet256.AddrFromBytes(msg.Datagram.Dst)
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

func (s *Server) Delete(ctx context.Context, req *DeleteReq) (*DeleteRes, error) {
	privateKey, err := serde.ParsePrivateKey(req.PrivateKey)
	if err != nil {
		return nil, err
	}
	if err := s.s.Delete(ctx, privateKey); err != nil {
		return nil, err
	}
	return &DeleteRes{}, nil
}
