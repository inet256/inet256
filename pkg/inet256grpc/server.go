package inet256grpc

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"io"

	"github.com/brendoncarroll/go-p2p"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ INET256Server = &Server{}

type Server struct {
	n *inet256.Node

	UnimplementedINET256Server
}

func NewServer(n *inet256.Node) *Server {
	return &Server{
		n: n,
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
	pubKey, err := s.n.LookupPublicKey(ctx, target)
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
	mtu := s.n.MTU(ctx, target)
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
	n := s.n.NewVirtual(privKey)
	defer func() {
		if err := n.Close(); err != nil {
			logrus.Error(err)
		}
	}()
	n.OnRecv(func(src, dst inet256.Addr, payload []byte) {
		if err := srv.Send(&ConnectMsg{
			Datagram: &Datagram{
				Src:     src[:],
				Dst:     dst[:],
				Payload: payload,
			},
		}); err != nil {
			logrus.Error(err)
		}
	})
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
		if err := n.Tell(ctx, dst, msg.Datagram.Payload); err != nil {
			return err
		}
	}
	return n.Close()
}

func (s *Server) findAddr(ctx context.Context, prefix []byte) (inet256.Addr, error) {
	return s.n.FindAddr(ctx, prefix, len(prefix)*8)
}
