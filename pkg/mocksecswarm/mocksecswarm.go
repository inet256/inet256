package mocksecswarm

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/swarmutil"
)

var _ p2p.SecureSwarm = &Swarm{}

type Swarm struct {
	privateKey p2p.PrivateKey
	localID    p2p.PeerID
	inner      p2p.Swarm

	onTell p2p.TellHandler

	keyCache sync.Map
}

func New(inner p2p.Swarm, privateKey p2p.PrivateKey) p2p.SecureSwarm {
	s := &Swarm{
		localID:    p2p.NewPeerID(privateKey.Public()),
		privateKey: privateKey,
		inner:      inner,
		onTell:     p2p.NoOpTellHandler,
	}
	s.inner.OnTell(s.fromBelow)
	return s
}

func (s *Swarm) Tell(ctx context.Context, addr p2p.Addr, data []byte) error {
	dst := addr.(Addr)
	pubKeyData := p2p.MarshalPublicKey(s.privateKey.Public())
	msg := message{
		SrcPublicKey: pubKeyData,
		Payload:      data,
	}
	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.inner.Tell(ctx, dst.Addr, msgData)
}

func (s *Swarm) OnTell(fn p2p.TellHandler) {
	swarmutil.AtomicSetTH(&s.onTell, fn)
}

func (s *Swarm) LookupPublicKey(ctx context.Context, target p2p.Addr) (p2p.PublicKey, error) {
	v, ok := s.keyCache.Load(target)
	if !ok {
		return nil, p2p.ErrPublicKeyNotFound
	}
	return v.(p2p.PublicKey), nil
}

func (s *Swarm) PublicKey() p2p.PublicKey {
	return s.privateKey.Public()
}

func (s *Swarm) Close() error {
	return s.inner.Close()
}

func (s *Swarm) LocalAddrs() []p2p.Addr {
	return []p2p.Addr{s.LocalAddr()}
}

func (s *Swarm) MTU(ctx context.Context, target p2p.Addr) int {
	dst := target.(Addr)
	return s.inner.MTU(ctx, dst.Addr) - 200
}

func (s *Swarm) LocalAddr() Addr {
	return Addr{
		ID:   s.localID,
		Addr: s.inner.LocalAddrs()[0],
	}
}

func (s *Swarm) ParseAddr(data []byte) (p2p.Addr, error) {
	parts := bytes.SplitN(data, []byte("@"), 2)
	id := p2p.PeerID{}
	if err := id.UnmarshalText(parts[0]); err != nil {
		return nil, err
	}
	innerAddr, err := s.inner.ParseAddr(parts[1])
	if err != nil {
		return nil, err
	}
	return Addr{
		ID:   id,
		Addr: innerAddr,
	}, nil
}

func (s *Swarm) fromBelow(msg *p2p.Message) {
	dst := s.LocalAddr()
	msg2 := message{}
	if err := json.Unmarshal(msg.Payload, &msg2); err != nil {
		panic(err)
	}
	srcPubKey, err := p2p.ParsePublicKey(msg2.SrcPublicKey)
	if err != nil {
		panic(err)
	}
	src := Addr{
		ID:   p2p.NewPeerID(srcPubKey),
		Addr: msg.Src,
	}
	s.keyCache.Store(src, srcPubKey)
	msg3 := p2p.Message{
		Src:     Addr{ID: src.ID, Addr: msg.Src},
		Dst:     Addr{ID: dst.ID, Addr: msg.Dst},
		Payload: msg2.Payload,
	}
	onTell := swarmutil.AtomicGetTH(&s.onTell)
	onTell(&msg3)
}

type message struct {
	SrcPublicKey []byte
	Payload      []byte
}
