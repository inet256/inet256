package inet256ipc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/inet256/inet256/pkg/inet256"
	"golang.org/x/sync/errgroup"
)

type serveNodeConfig struct {
	KeepAliveInterval time.Duration
	KeepAliveTimeout  time.Duration
}

type ServeNodeOption func(c *serveNodeConfig)

// ServeNode reads and writes packets from a Framer, using the node to
// send network data and answer requests.
// If the context is cancelled ServeNode returns nil.
func ServeNode(ctx context.Context, node inet256.Node, transport SendReceiver, opts ...ServeNodeOption) error {
	config := serveNodeConfig{
		KeepAliveInterval: DefaultKeepAliveInterval,
		KeepAliveTimeout:  DefaultKeepAliveTimeout,
	}
	ctx, cf := context.WithCancel(ctx)
	defer cf()
	eg, ctx := errgroup.WithContext(ctx)
	pool := newBufPool()
	var gotOne atomic.Bool
	// read loop
	eg.Go(func() error {
		for {
			if err := transport.Receive(ctx, func(data []byte) {
				// handle message
				if err := func() error {
					msg, err := AsMessage(data, true)
					if err != nil {
						return err
					}
					switch msg.GetType() {
					case MT_Data:
						dm := msg.DataMsg()
						if err := node.Send(ctx, dm.Addr, dm.Payload); err != nil {
							return err
						}
					case MT_KeepAlive:
					default:
						// copy message
						reqBuf := pool.Acquire()
						msg, err := AsMessage(reqBuf[:copy(reqBuf[:], data)], true)
						if err != nil {
							panic(err) // This already happens above
						}
						// spawn goroutine for request
						eg.Go(func() error {
							defer pool.Release(reqBuf)

							resBuf := pool.Acquire()
							defer pool.Release(resBuf)
							resn, err := handleAsk(ctx, node, msg, resBuf[:])
							if err != nil {
								logctx.Errorf(ctx, "handling message: %v", err)
							}
							if err := transport.Send(ctx, resBuf[:resn]); err != nil {
								logctx.Errorf(ctx, "writing response frame: %v", err)
							}
							return nil
						})
					}
					return nil
				}(); err != nil {
					logctx.Errorf(ctx, "handling message: %v", err)
				}
			}); err != nil {
				return err
			}
			gotOne.Store(true)
		}
	})
	eg.Go(func() error {
		for {
			var sendErr error
			if err := node.Receive(ctx, func(msg inet256.Message) {
				buf := pool.Acquire()
				defer pool.Release(buf)
				n := WriteDataMessage(buf[:], msg.Src, msg.Payload)
				transport.Send(ctx, buf[:n])
			}); err != nil {
				return err
			}
			if sendErr != nil {
				return sendErr
			}
		}
	})
	eg.Go(func() error {
		ticker := time.NewTicker(config.KeepAliveInterval)
		defer ticker.Stop()
		for {
			buf := pool.Acquire()
			n := WriteKeepAlive(buf[:])
			if err := transport.Send(ctx, buf[:n]); err != nil {
				pool.Release(buf)
				return err
			}
			pool.Release(buf)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	})
	eg.Go(func() error {
		timer := time.NewTimer(config.KeepAliveTimeout)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				if !gotOne.Load() {
					return errors.New("inet256ipc: server missed keep alive from client")
				}
				gotOne.Store(false)
				timer.Reset(config.KeepAliveTimeout)
			}
		}
	})
	err := eg.Wait()
	if errors.Is(err, ctx.Err()) {
		err = nil
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return err
}

func handleAsk(ctx context.Context, node inet256.Node, msg Message, resBuf []byte) (n int, _ error) {
	switch msg.GetType() {
	case MT_MTU:
		req, err := msg.MTUReq()
		if err != nil {
			return 0, err
		}
		mtu := node.MTU(ctx, req.Target)
		if err != nil {
			return 0, err
		}
		n = WriteAskMessage(resBuf, msg.GetRequestID(), MT_MTU, MTURes{
			MTU: mtu,
		})
	case MT_FindAddr:
		req, err := msg.FindAddrReq()
		if err != nil {
			return 0, err
		}
		addr, err := node.FindAddr(ctx, req.Prefix, req.Nbits)
		if err != nil {
			n = WriteError[FindAddrRes](resBuf, msg.GetRequestID(), MT_FindAddr, err)
		} else {
			n = WriteSuccess(resBuf, msg.GetRequestID(), MT_FindAddr, FindAddrRes{
				Addr: addr,
			})
		}
	case MT_PublicKey:
		req, err := msg.LookupPublicKeyReq()
		if err != nil {
			return 0, err
		}
		pubKey, err := node.LookupPublicKey(ctx, req.Target)
		if err != nil {
			n = WriteError[LookupPublicKeyRes](resBuf, msg.GetRequestID(), MT_PublicKey, err)
		} else {
			n = WriteSuccess(resBuf, msg.GetRequestID(), MT_PublicKey, LookupPublicKeyRes{
				PublicKey: inet256.MarshalPublicKey(nil, pubKey),
			})
		}
	case MT_KeepAlive:
	default:
		return 0, fmt.Errorf("unrecognized message type %v", msg.GetType())
	}
	return n, nil
}
