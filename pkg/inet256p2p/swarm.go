package inet256p2p

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/client/go_client/inet256client"
	"github.com/inet256/inet256/pkg/inet256"
)

var _ p2p.SecureAskSwarm = &Swarm{}
var _ peerswarm.AskSwarm = &Swarm{}

type Swarm struct {
	publicKey    p2p.PublicKey
	inet256Swarm peerswarm.Swarm
	asker        *asker
}

func NewSwarm(endpoint string, privateKey p2p.PrivateKey) (peerswarm.AskSwarm, error) {
	client, err := inet256client.NewNode(endpoint, privateKey)
	if err != nil {
		return nil, err
	}
	inetsw := inet256.SwarmFromNetwork(client, privateKey.Public())
	s := &Swarm{
		publicKey:    privateKey.Public(),
		inet256Swarm: inetsw,
		asker:        newAsker(inetsw),
	}
	return s, nil
}

func (s *Swarm) Tell(ctx context.Context, dst p2p.Addr, data []byte) error {
	return s.asker.Tell(ctx, dst, data)
}

func (s *Swarm) TellPeer(ctx context.Context, dst p2p.PeerID, data []byte) error {
	return s.asker.Tell(ctx, dst, data)
}

func (s *Swarm) OnTell(fn p2p.TellHandler) {
	s.asker.OnTell(fn)
}

func (s *Swarm) Ask(ctx context.Context, dst p2p.Addr, data []byte) ([]byte, error) {
	return s.asker.Ask(ctx, dst, data)
}

func (s *Swarm) AskPeer(ctx context.Context, dst p2p.PeerID, data []byte) ([]byte, error) {
	return s.asker.Ask(ctx, dst, data)
}

func (s *Swarm) OnAsk(fn p2p.AskHandler) {
	s.asker.OnAsk(fn)
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
	return s.asker.MTU(ctx, target.(p2p.PeerID))
}

func (s *Swarm) Close() error {
	return s.asker.Close()
}

func (s *Swarm) ParseAddr(data []byte) (p2p.Addr, error) {
	id := p2p.PeerID{}
	if err := id.UnmarshalText(data); err != nil {
		return nil, err
	}
	return id, nil
}