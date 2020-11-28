package inet256

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/x509"
	"io"

	"github.com/brendoncarroll/go-p2p"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ inet256grpc.INET256Server = &Server{}

type Server struct {
	n *Node

	inet256grpc.UnimplementedINET256Server
}

func NewServer(n *Node) *Server {
	return &Server{
		n: n,
	}
}

func (s *Server) GenerateKey(ctx context.Context, _ *empty.Empty) (*inet256grpc.GenerateKeyRes, error) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}
	keyData, err := MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	return &inet256grpc.GenerateKeyRes{
		PrivateKey: keyData,
	}, nil
}

func (s *Server) LookupSelf(ctx context.Context, req *inet256grpc.LookupSelfReq) (*inet256grpc.PeerInfo, error) {
	privKey, err := ParsePrivateKey(req.PrivateKey)
	if err != nil {
		return nil, err
	}
	privKey2, ok := privKey.(interface{ Public() crypto.PublicKey })
	if !ok {
		return nil, errors.Errorf("unsupported key")
	}
	pubKey := privKey2.Public()
	peerID := p2p.NewPeerID(pubKey)
	return &inet256grpc.PeerInfo{
		Addr:      peerID[:],
		PublicKey: p2p.MarshalPublicKey(pubKey),
	}, nil
}

func (s *Server) Lookup(ctx context.Context, req *inet256grpc.LookupReq) (*inet256grpc.PeerInfo, error) {
	target := Addr{}
	copy(target[:], req.TargetAddr)
	nwk := s.n.whichNetwork(ctx, target)
	if nwk == nil {
		return nil, ErrAddrUnreachable
	}
	pubKey, err := nwk.LookupPublicKey(ctx, target)
	if err != nil {
		return nil, err
	}
	return &inet256grpc.PeerInfo{
		Addr:      target[:],
		PublicKey: p2p.MarshalPublicKey(pubKey),
	}, nil
}

func (s *Server) Connect(srv inet256grpc.INET256_ConnectServer) error {
	msg, err := srv.Recv()
	if err != nil {
		return err
	}
	privKey, err := ParsePrivateKey(msg.PrivateKey)
	if err != nil {
		return err
	}
	n := s.n.NewVirtual(privKey)
	defer func() {
		if err := n.Close(); err != nil {
			logrus.Error(err)
		}
	}()
	n.OnRecv(func(src, dst Addr, payload []byte) {
		if err := srv.Send(&inet256grpc.ConnectMsg{
			Datagram: &inet256grpc.Datagram{
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
		dst := Addr{}
		copy(dst[:], msg.Datagram.Dst)
		if err := n.Tell(ctx, dst, msg.Datagram.Payload); err != nil {
			return err
		}
	}
	return n.Close()
}

func ParsePrivateKey(data []byte) (p2p.PrivateKey, error) {
	privKey, err := x509.ParsePKCS8PrivateKey(data)
	if err != nil {
		return nil, err
	}
	privKey2, ok := privKey.(p2p.PrivateKey)
	if !ok {
		return nil, errors.Errorf("unsupported private key type")
	}
	return privKey2, nil
}

func MarshalPrivateKey(privKey p2p.PrivateKey) ([]byte, error) {
	return x509.MarshalPKCS8PrivateKey(privKey)
}
