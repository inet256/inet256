// package router provides a template for implementing Networks
// Network implements a mesh256.Network in terms of a Router
package routers

import (
	"context"
	"runtime"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/futures"
	"github.com/brendoncarroll/go-tai64"
	"github.com/brendoncarroll/stdctx/logctx"

	"github.com/inet256/inet256/internal/netutil"
	"github.com/inet256/inet256/internal/retry"
	"github.com/inet256/inet256/pkg/bitstr"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/maybe"
	"github.com/inet256/inet256/pkg/mesh256"
)

type SendFunc = func(inet256.Addr, p2p.IOVec)

type AboveContext struct {
	context.Context

	Send SendFunc
}

// BelowContext is passed to handle below
type BelowContext struct {
	context.Context

	Now         tai64.TAI64
	Send        SendFunc
	OnData      func(src inet256.Addr, data []byte)
	OnAddr      func(inet256.Addr) bool
	OnMTU       func(inet256.Addr, int) bool
	OnPublicKey func(inet256.Addr, inet256.PublicKey) bool
}

type RouterParams struct {
	PrivateKey   inet256.PrivateKey
	Peers        mesh256.PeerSet
	GetPublicKey func(inet256.Addr) inet256.PublicKey
	Now          time.Time
}

// Router contains logic for routing messages throughout the network
// Routers do not maintain any backgroud goroutines or allocate any additional resources
// If all references to a router are lost, resources MUST not leak.
type Router interface {
	// Reset is called to clear the state of the router
	Reset(RouterParams)

	// HandleAbove handles a message from the application
	HandleAbove(ctx *AboveContext, to inet256.Addr, data p2p.IOVec) bool
	// HandleBelow handles a message from the network.
	HandleBelow(ctx *BelowContext, from inet256.Addr, data []byte)
	// Heartbeat is called periodically with the current time.
	Heartbeat(ctx *AboveContext, now time.Time)

	// FindAddr looks up an address
	FindAddr(ctx *AboveContext, prefix []byte, nbits int) maybe.Maybe[inet256.Addr]
	// LookupPublicKey looks up a public key
	LookupPublicKey(ctx *AboveContext, target inet256.Addr) maybe.Maybe[inet256.PublicKey]
	// MTU
	MTU(ctx *AboveContext, target inet256.Addr) maybe.Maybe[int]
}

var _ mesh256.Network = &Network{}

type Network struct {
	params    mesh256.NetworkParams
	publicKey inet256.PublicKey
	localID   inet256.Addr

	router       Router
	tellHub      netutil.TellHub
	sg           netutil.ServiceGroup
	findAddr     *futures.Store[bitstr.String, inet256.Addr]
	lookupPubKey *futures.Store[inet256.Addr, inet256.PublicKey]
}

// NewNetwork returns a network implemented in terms of a router.
func NewNetwork(params mesh256.NetworkParams, router Router, hbPeriod time.Duration) *Network {
	publicKey := params.PrivateKey.Public()
	n := &Network{
		params:    params,
		publicKey: publicKey,
		localID:   inet256.NewAddr(publicKey),

		router:       router,
		tellHub:      netutil.NewTellHub(),
		findAddr:     futures.NewStore[bitstr.String, inet256.Addr](),
		lookupPubKey: futures.NewStore[inet256.Addr, inet256.PublicKey](),
	}
	router.Reset(RouterParams{
		PrivateKey:   params.PrivateKey,
		Peers:        params.Peers,
		GetPublicKey: n.getPublicKey,
		Now:          time.Now(),
	})
	numWorkers := 1 + runtime.GOMAXPROCS(0)
	for i := 0; i < numWorkers; i++ {
		n.sg.Go(n.readLoop)
	}
	n.sg.Go(func(ctx context.Context) error {
		return n.heartbeatLoop(ctx, hbPeriod)
	})
	return n
}

func (n *Network) Tell(ctx context.Context, dst inet256.Addr, data p2p.IOVec) error {
	ctx, cf := context.WithTimeout(ctx, 3*time.Second)
	defer cf()
	return retry.Retry(ctx, func() (retErr error) {
		ctx := &AboveContext{
			Context: ctx,
			Send: func(dst inet256.Addr, data p2p.IOVec) {
				if err := n.params.Swarm.Tell(ctx, dst, data); err != nil {
					retErr = err
				}
			},
		}
		ok := n.router.HandleAbove(ctx, dst, data)
		if ok {
			return nil
		}
		if retErr != nil {
			return retErr
		}
		return inet256.ErrAddrUnreachable{Addr: dst}
	})
}

func (n *Network) Receive(ctx context.Context, fn func(p2p.Message[inet256.Addr])) error {
	return n.tellHub.Receive(ctx, fn)
}

func (n *Network) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	send := func(dst inet256.Addr, data p2p.IOVec) {
		if err := n.params.Swarm.Tell(ctx, dst, data); err != nil {
			logctx.Warnln(ctx, err)
		}
	}
	bs := bitstr.FromSource(bitstr.BytesMSB{Bytes: prefix, End: nbits})
	fut, created := n.findAddr.GetOrCreate(bs)
	if created {
		defer n.findAddr.Delete(bs, fut)
	}
	return retry.RetryRet1(ctx, func() (inet256.Addr, error) {
		actx := &AboveContext{Context: ctx, Send: send}
		maybeAddr := n.router.FindAddr(actx, prefix, nbits)
		if maybeAddr.Ok {
			fut.Succeed(maybeAddr.X)
		}
		return futures.Await[inet256.Addr](ctx, fut)
	}, retry.WithBackoff(retry.NewConstantBackoff(200*time.Millisecond)))
}

func (n *Network) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	if n.params.Peers.Contains(target) {
		return n.params.Swarm.LookupPublicKey(ctx, target)
	}
	send := func(dst inet256.Addr, data p2p.IOVec) {
		if err := n.params.Swarm.Tell(ctx, dst, data); err != nil {
			logctx.Warnln(ctx, err)
		}
	}
	fut, created := n.lookupPubKey.GetOrCreate(target)
	if created {
		defer n.lookupPubKey.Delete(target, fut)
	}
	return retry.RetryRet1(ctx, func() (inet256.PublicKey, error) {
		actx := &AboveContext{Context: ctx, Send: send}
		maybePub := n.router.LookupPublicKey(actx, target)
		if maybePub.Ok {
			fut.Succeed(maybePub.X)
		}
		return futures.Await[inet256.PublicKey](ctx, fut)
	}, retry.WithBackoff(retry.NewConstantBackoff(200*time.Millisecond)))
}

func (n *Network) LocalAddr() inet256.Addr {
	return n.localID
}

func (n *Network) MTU(ctx context.Context, target inet256.Addr) int {
	return inet256.MaxMTU
}

func (n *Network) PublicKey() inet256.PublicKey {
	return n.publicKey
}

func (n *Network) Bootstrap(ctx context.Context) error {
	return nil
}

func (n *Network) Close() error {
	n.tellHub.CloseWithError(inet256.ErrClosed)
	return n.sg.Stop()
}

func (n *Network) readLoop(ctx context.Context) error {
	bctx := &BelowContext{
		Context: ctx,
		Send: func(dst inet256.Addr, data p2p.IOVec) {
			if err := n.params.Swarm.Tell(ctx, dst, data); err != nil {
				logctx.Warnln(ctx, err)
			}
		},
		OnData: func(src inet256.Addr, data []byte) {
			if err := n.tellHub.Deliver(ctx, p2p.Message[inet256.Addr]{
				Src:     src,
				Dst:     n.localID,
				Payload: data,
			}); err != nil {
				logctx.Warnln(ctx, err)
			}
		},
		OnPublicKey: n.handlePublicKey,
		OnAddr:      n.handleAddr,
		OnMTU: func(target inet256.Addr, mtu int) bool {
			return false
		},
	}
	var msg p2p.Message[inet256.Addr]
	for {
		if err := p2p.Receive[inet256.Addr](ctx, n.params.Swarm, &msg); err != nil {
			return err
		}
		n.router.HandleBelow(bctx, msg.Src, msg.Payload)
	}
}

// handleAddr handles an address found by the router in response to a FindAddr request
// it returns true if the target fulfilled an active request.
func (n *Network) handleAddr(target inet256.Addr) (ret bool) {
	bs := bitstr.FromSource(bitstr.BytesMSB{Bytes: target[:], End: 256})
	n.findAddr.ForEach(func(prefix bitstr.String, fut *futures.Promise[inet256.Addr]) bool {
		if bitstr.HasPrefix(bs, prefix) {
			if fut.Succeed(target) {
				ret = true
			}
		}
		return true // continue iteration
	})
	return ret
}

// handlePublicKey handle a public key found by the router in response to a LookupPublicKey.
// it returns true if the target fulfilled an active request.
func (n *Network) handlePublicKey(target inet256.Addr, publicKey inet256.PublicKey) bool {
	if fut := n.lookupPubKey.Get(target); fut != nil {
		return fut.Succeed(publicKey)
	}
	return false
}

func (n *Network) heartbeatLoop(ctx context.Context, d time.Duration) error {
	if d == 0 {
		return nil
	}
	send := func(dst inet256.Addr, data p2p.IOVec) {
		if err := n.params.Swarm.Tell(ctx, dst, data); err != nil {
			logctx.Warnln(ctx, err)
		}
	}
	tick := time.NewTicker(d)
	now := time.Now().UTC()
	for {
		n.router.Heartbeat(&AboveContext{Context: ctx, Send: send}, now)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-tick.C:
			now = t.UTC()
		}
	}
}

func (n *Network) getPublicKey(x inet256.Addr) inet256.PublicKey {
	ctx, cf := context.WithTimeout(context.Background(), time.Millisecond)
	defer cf()
	pubKey, err := n.params.Swarm.LookupPublicKey(ctx, x)
	if err != nil {
		// TODO
		panic(err)
	}
	return pubKey
}
