// package neteng provides a template for implementing Networks
// Network implements a mesh256.Network in terms of a Router
package neteng

import (
	"context"
	"runtime"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/futures"

	"github.com/inet256/inet256/pkg/bitstr"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/inet256/inet256/pkg/retry"
)

// SendFunc is the type of functions used to send data to peers
type SendFunc = func(dst inet256.Addr, data p2p.IOVec)

// DeliverFunc is the type of functions used to deliver data to the application
type DeliverFunc = func(src inet256.Addr, data []byte)

// InfoFunc is the type of functions used to deliver information about peers
type InfoFunc = func(inet256.Addr, inet256.PublicKey)

// PublicKeyFunc is the type of functinos used to get the public key for an ID
type PublicKeyFunc = func(inet256.Addr) inet256.PublicKey

// Router contains logic for routing messages throughout the network
// Routers do not maintain any backgroud goroutines or allocate any additional resources
// If all references to a router are lost, resources MUST not leak.
type Router interface {
	// Reset is called to clear the state of the router
	Reset(privateKey inet256.PrivateKey, peers mesh256.PeerSet, getPublicKey PublicKeyFunc, now time.Time)
	// HandleAbove handles a message from the application
	HandleAbove(to inet256.Addr, data p2p.IOVec, send SendFunc) bool
	// HandleBelow handles a message from the network.
	HandleBelow(from inet256.Addr, data []byte, send SendFunc, deliver DeliverFunc, info InfoFunc)
	// Heartbeat is called periodically with the current time.
	Heartbeat(now time.Time, send SendFunc)

	// FindAddr looks up an address
	FindAddr(send SendFunc, info InfoFunc, prefix []byte, nbits int)
	// LookupPublicKey looks up a public key
	LookupPublicKey(send SendFunc, info InfoFunc, target inet256.Addr)
}

var _ mesh256.Network = &Network{}

type Network struct {
	params    mesh256.NetworkParams
	log       mesh256.Logger
	publicKey inet256.PublicKey
	localID   inet256.Addr

	router       Router
	tellHub      netutil.TellHub
	sg           netutil.ServiceGroup
	findAddr     *futures.Store[bitstr.String, inet256.Addr]
	lookupPubKey *futures.Store[inet256.Addr, inet256.PublicKey]
}

func New(params mesh256.NetworkParams, router Router, hbPeriod time.Duration) *Network {
	publicKey := params.PrivateKey.Public()
	n := &Network{
		params:    params,
		log:       params.Logger,
		publicKey: publicKey,
		localID:   inet256.NewAddr(publicKey),

		router:       router,
		tellHub:      *netutil.NewTellHub(),
		findAddr:     futures.NewStore[bitstr.String, inet256.Addr](),
		lookupPubKey: futures.NewStore[inet256.Addr, inet256.PublicKey](),
	}
	router.Reset(params.PrivateKey, params.Peers, n.getPublicKey, time.Now())
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
		ok := n.router.HandleAbove(dst, data, func(dst inet256.Addr, data p2p.IOVec) {
			if err := n.params.Swarm.Tell(ctx, dst, data); err != nil {
				retErr = err
			}
		})
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
			n.log.Warn(err)
		}
	}
	bs := bitstr.FromSource(bitstr.BytesMSB{Bytes: prefix, End: nbits})
	fut, created := n.findAddr.GetOrCreate(bs)
	if created {
		defer n.findAddr.Delete(bs, fut)
	}
	return retry.RetryRet1(ctx, func() (inet256.Addr, error) {
		n.router.FindAddr(send, n.handleInfo, prefix, nbits)
		return futures.Await[inet256.Addr](ctx, fut)
	}, retry.WithBackoff(retry.NewConstantBackoff(200*time.Millisecond)))
}

func (n *Network) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	if n.params.Peers.Contains(target) {
		return n.params.Swarm.LookupPublicKey(ctx, target)
	}
	send := func(dst inet256.Addr, data p2p.IOVec) {
		if err := n.params.Swarm.Tell(ctx, dst, data); err != nil {
			n.log.Warn(err)
		}
	}
	fut, created := n.lookupPubKey.GetOrCreate(target)
	if created {
		defer n.lookupPubKey.Delete(target, fut)
	}
	return retry.RetryRet1(ctx, func() (inet256.PublicKey, error) {
		n.router.LookupPublicKey(send, n.handleInfo, target)
		return futures.Await[inet256.PublicKey](ctx, fut)
	}, retry.WithBackoff(retry.NewConstantBackoff(200*time.Millisecond)))
}

func (n *Network) LocalAddr() inet256.Addr {
	return n.localID
}

func (n *Network) MTU(ctx context.Context, target inet256.Addr) int {
	return inet256.MinMTU
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
	send := func(dst inet256.Addr, data p2p.IOVec) {
		if err := n.params.Swarm.Tell(ctx, dst, data); err != nil {
			n.log.Warn(err)
		}
	}
	deliver := func(src inet256.Addr, data []byte) {
		if err := n.tellHub.Deliver(ctx, p2p.Message[inet256.Addr]{
			Src:     src,
			Dst:     n.localID,
			Payload: data,
		}); err != nil {
			n.log.Warn(err)
		}
	}
	var msg p2p.Message[inet256.Addr]
	for {
		if err := p2p.Receive[inet256.Addr](ctx, n.params.Swarm, &msg); err != nil {
			return err
		}
		n.router.HandleBelow(msg.Src, msg.Payload, send, deliver, n.handleInfo)
	}
}

func (n *Network) handleInfo(target inet256.Addr, publicKey inet256.PublicKey) {
	n.handlePublicKey(target, publicKey)
	n.handleAddr(target)
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
			n.log.Warn(err)
		}
	}
	tick := time.NewTicker(d)
	now := time.Now().UTC()
	for {
		n.router.Heartbeat(now, send)
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
