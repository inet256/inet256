package inet256p2p

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/client/go_client/inet256client"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
)

var _ p2p.SecureAskSwarm = &Swarm{}

type Swarm struct {
	publicKey    p2p.PublicKey
	inet256Swarm p2p.SecureSwarm
	asker        *asker
}

func NewSwarm(endpoint string, privateKey p2p.PrivateKey) (p2p.SecureAskSwarm, error) {
	client, err := inet256client.NewNode(endpoint, privateKey)
	if err != nil {
		return nil, err
	}
	inetsw := inet256srv.SwarmFromNetwork(client, privateKey.Public())
	s := &Swarm{
		publicKey:    privateKey.Public(),
		inet256Swarm: inetsw,
		asker:        newAsker(inetsw),
	}
	return s, nil
}

func (s *Swarm) Tell(ctx context.Context, dst p2p.Addr, data p2p.IOVec) error {
	return s.asker.Tell(ctx, dst, data)
}

func (s *Swarm) Recv(ctx context.Context, src, dst *p2p.Addr, buf []byte) (int, error) {
	return s.asker.Recv(ctx, src, dst, buf)
}

func (s *Swarm) Ask(ctx context.Context, resp []byte, dst p2p.Addr, data p2p.IOVec) (int, error) {
	return s.asker.Ask(ctx, resp, dst.(inet256.Addr), data)
}

func (s *Swarm) ServeAsk(ctx context.Context, fn p2p.AskHandler) error {
	return s.asker.ServeAsk(ctx, fn)
}

func (s *Swarm) LocalAddrs() []p2p.Addr {
	return s.inet256Swarm.LocalAddrs()
}

func (s *Swarm) PublicKey() p2p.PublicKey {
	return s.inet256Swarm.PublicKey()
}

func (s *Swarm) LookupPublicKey(ctx context.Context, target p2p.Addr) (p2p.PublicKey, error) {
	return s.inet256Swarm.LookupPublicKey(ctx, target)
}

func (s *Swarm) MTU(ctx context.Context, target p2p.Addr) int {
	return s.asker.MTU(ctx, target.(inet256.Addr))
}

func (s *Swarm) MaxIncomingSize() int {
	return inet256.MaxMTU
}

func (s *Swarm) Close() error {
	s.asker.Close()
	return s.inet256Swarm.Close()
}

func (s *Swarm) ParseAddr(data []byte) (p2p.Addr, error) {
	return s.inet256Swarm.ParseAddr(data)
}
