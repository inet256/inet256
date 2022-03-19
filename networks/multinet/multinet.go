package multinet

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/p2pmux"
	"github.com/brendoncarroll/go-p2p/s/fragswarm"
	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/inet256/inet256/pkg/p2padapter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Spec map[NetworkCode]networks.Factory

func NewFactory(spec Spec) networks.Factory {
	return func(params networks.Params) networks.Network {
		mux := p2pmux.NewUint64SecureMux[inet256.Addr](p2padapter.P2PFromINET256(params.Swarm))
		// create multi network
		nwks := make([]networks.Network, 0, len(spec))
		for code, factory := range spec {
			s := mux.Open(binary.BigEndian.Uint64(code[:]))
			s = fragswarm.NewSecure(s, networks.TransportMTU)
			nw := factory(networks.Params{
				PrivateKey: params.PrivateKey,
				Swarm:      p2padapter.INET256FromP2P(s),
				Peers:      params.Peers,
				Logger:     logrus.StandardLogger(),
			})
			nwks = append(nwks, nw)
		}
		return New(nwks...)
	}
}

type NetworkCode = [8]byte

// Network presents multiple networks as a single network
type Network struct {
	networks []networks.Network
	addrMap  sync.Map
	sg       *netutil.ServiceGroup
	hub      *netutil.TellHub
}

func New(nwks ...networks.Network) *Network {
	for i := 0; i < len(nwks)-1; i++ {
		if nwks[i].LocalAddr() != nwks[i+1].LocalAddr() {
			panic("network addresses do not match")
		}
	}
	hub := netutil.NewTellHub()
	sg := &netutil.ServiceGroup{}
	for i := range nwks {
		sg.Go(func(ctx context.Context) error {
			for {
				if err := nwks[i].Receive(ctx, func(m p2p.Message[inet256.Addr]) {
					hub.Deliver(ctx, m)
				}); err != nil {
					return err
				}
			}
		})
	}
	return &Network{
		networks: nwks,
		sg:       sg,
		hub:      hub,
	}
}

func (mn *Network) Tell(ctx context.Context, dst inet256.Addr, v p2p.IOVec) error {
	network, err := mn.whichNetwork(ctx, dst)
	if err != nil {
		return err
	}
	return network.Tell(ctx, dst, v)
}

func (mn *Network) Receive(ctx context.Context, fn func(p2p.Message[inet256.Addr])) error {
	return mn.hub.Receive(ctx, fn)
}

func (mn *Network) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	addr, _, err := mn.addrWithPrefix(ctx, prefix, nbits)
	return addr, err
}

func (mn *Network) LookupPublicKey(ctx context.Context, target inet256.Addr) (p2p.PublicKey, error) {
	network, err := mn.whichNetwork(ctx, target)
	if err != nil {
		return nil, err
	}
	return network.LookupPublicKey(ctx, target)
}

func (mn *Network) MTU(ctx context.Context, target inet256.Addr) int {
	network, err := mn.whichNetwork(ctx, target)
	if err != nil {
		return inet256.MinMTU
	}
	return network.MTU(ctx, target)
}

func (mn *Network) PublicKey() inet256.PublicKey {
	return mn.networks[0].PublicKey()
}

func (mn *Network) LocalAddr() inet256.Addr {
	return mn.networks[0].LocalAddr()
}

func (mn *Network) Bootstrap(ctx context.Context) error {
	eg := errgroup.Group{}
	for _, n := range mn.networks {
		n := n
		eg.Go(func() error {
			return n.Bootstrap(ctx)
		})
	}
	return eg.Wait()
}

func (mn *Network) Close() (retErr error) {
	var el netutil.ErrList
	el.Add(mn.sg.Stop())
	mn.hub.CloseWithError(inet256.ErrClosed)
	for _, n := range mn.networks {
		el.Add(n.Close())
	}
	return el.Err()
}

func (mn *Network) addrWithPrefix(ctx context.Context, prefix []byte, nbits int) (addr inet256.Addr, network networks.Network, err error) {
	var cf context.CancelFunc
	if deadline, ok := ctx.Deadline(); ok {
		now := time.Now()
		ctx, cf = context.WithTimeout(ctx, deadline.Sub(now)/3)
	} else {
		ctx, cf = context.WithCancel(ctx)
	}
	defer cf()

	addrs := make([]inet256.Addr, len(mn.networks))
	errs := make([]error, len(mn.networks))
	wg := sync.WaitGroup{}
	wg.Add(len(mn.networks))
	for i, network := range mn.networks[:] {
		i := i
		network := network
		go func() {
			addrs[i], errs[i] = network.FindAddr(ctx, prefix, nbits)
			if errs[i] == nil {
				cf()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	for i, addr := range addrs {
		if errs[i] == nil {
			return addr, mn.networks[i], nil
		}
	}
	return inet256.Addr{}, nil, fmt.Errorf("errors occurred %v", errs)
}

func (mn *Network) whichNetwork(ctx context.Context, addr inet256.Addr) (networks.Network, error) {
	x, exists := mn.addrMap.Load(addr)
	if exists {
		return x.(networks.Network), nil
	}
	_, network, err := mn.addrWithPrefix(ctx, addr[:], len(addr)*8)
	if err != nil {
		return nil, errors.Wrap(err, "selecting network")
	}
	mn.addrMap.Store(addr, network)
	return network, nil
}
