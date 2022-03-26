package centraldisco

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-tai64"
	"github.com/golang/protobuf/proto"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/discovery/centraldisco/internal"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type peerState struct {
	Timestamp time.Time
	TTL       time.Duration

	PublicKey []byte
	Announce  *internal.Announce
	Sig       []byte
}

type Server struct {
	log    *logrus.Logger
	parser discovery.AddrParser

	mu sync.RWMutex
	m  map[inet256.Addr]*peerState

	internal.UnimplementedDiscoveryServer
}

func RunServer(l net.Listener, s *Server) error {
	gs := grpc.NewServer()
	internal.RegisterDiscoveryServer(gs, s)
	return gs.Serve(l)
}

func NewServer(log *logrus.Logger, parser discovery.AddrParser) *Server {
	return &Server{
		log:    log,
		parser: parser,
		m:      make(map[inet256.Addr]*peerState),
	}
}

func (s *Server) Announce(ctx context.Context, req *internal.AnnounceReq) (*internal.AnnounceRes, error) {
	pubKey, err := inet256.ParsePublicKey(req.PublicKey)
	if err != nil {
		return nil, err
	}
	if err := p2p.Verify(pubKey, purposeAnnounce, req.Announce, req.Sig); err != nil {
		return nil, err
	}
	var x internal.Announce
	if err := proto.Unmarshal(req.Announce, &x); err != nil {
		return nil, err
	}
	timeNext, err := tai64.Parse(x.Tai64)
	if err != nil {
		return nil, err
	}
	for _, endpoint := range x.Endpoints {
		if _, err := s.parser([]byte(endpoint)); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "error parsing endpoint: %v", err)
		}
	}
	addr := inet256.NewAddr(pubKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, exists := s.m[addr]; exists {
		if !timeNext.GoTime().After(prev.Timestamp) {
			return nil, status.Errorf(codes.FailedPrecondition, "timestamp %v <= %v", timeNext.GoTime(), prev.Timestamp)
		}
	}
	s.m[addr] = &peerState{
		Timestamp: timeNext.GoTime(),
		TTL:       time.Duration(x.TtlSeconds) * time.Second,
		Announce:  &x,
		Sig:       req.Sig,
		PublicKey: req.PublicKey,
	}
	return &internal.AnnounceRes{}, nil
}

func (s *Server) Find(ctx context.Context, req *internal.FindReq) (*internal.FindRes, error) {
	if len(req.GetTarget()) < len(inet256.Addr{}) {
		return nil, status.Errorf(codes.InvalidArgument, "target too short to be INET256 address")
	}
	addr := inet256.AddrFromBytes(req.GetTarget())
	x := s.get(addr)
	if x == nil {
		return nil, status.Errorf(codes.NotFound, "no announce found for peer %v", addr)
	}
	announceBytes, err := proto.Marshal(x.Announce)
	if err != nil {
		panic(err)
	}
	return &internal.FindRes{
		PublicKey: x.PublicKey,
		Announce:  announceBytes,
		Sig:       x.Sig,
	}, nil
}

func (s *Server) get(x inet256.Addr) *peerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[x]
}
