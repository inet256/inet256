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
func ServeNode(ctx context.Context, node inet256.Node, framer Framer, opts ...ServeNodeOption) error {
	config := serveNodeConfig{
		KeepAliveInterval: DefaultKeepAliveInterval,
		KeepAliveTimeout:  DefaultKeepAliveTimeout,
	}
	ctx, cf := context.WithCancel(ctx)
	defer cf()
	eg, ctx := errgroup.WithContext(ctx)
	pool := newFramePool()
	var gotOne atomic.Bool
	// read loop
	eg.Go(func() error {
		for {
			fr := pool.Acquire()
			if err := framer.ReadFrame(ctx, fr); err != nil {
				return err
			}
			gotOne.Store(true)
			// handle message
			if err := func() error {
				msg, err := AsMessage(fr.Body(), true)
				if err != nil {
					return err
				}
				switch msg.GetType() {
				case MT_Data:
					dm := msg.DataMsg()
					if err := node.Send(ctx, dm.Addr, dm.Payload); err != nil {
						return err
					}
					pool.Release(fr)
				case MT_KeepAlive:
				default:
					// spawn goroutine for request
					eg.Go(func() error {
						defer pool.Release(fr)
						resFr := pool.Acquire()
						defer pool.Release(resFr)
						if err := handleAsk(ctx, node, msg, resFr); err != nil {
							logctx.Errorf(ctx, "handling message: %v", err)
						}
						if err := framer.WriteFrame(ctx, resFr); err != nil {
							logctx.Errorf(ctx, "writing response frame: %v", err)
						}
						return nil
					})
				}
				return nil
			}(); err != nil {
				logctx.Errorf(ctx, "handling message: %v", err)
			}
		}
	})
	eg.Go(func() error {
		for {
			var sendErr error
			if err := node.Receive(ctx, func(msg inet256.Message) {
				fr := pool.Acquire()
				defer pool.Release(fr)
				WriteDataMessage(fr, msg.Src, msg.Payload)
				if err := framer.WriteFrame(ctx, fr); err != nil {
					sendErr = err
				}
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
			fr := pool.Acquire()
			WriteKeepAlive(fr)
			if err := framer.WriteFrame(ctx, fr); err != nil {
				return err
			}
			pool.Release(fr)

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

func handleAsk(ctx context.Context, node inet256.Node, msg Message, resFr *Frame) error {
	switch msg.GetType() {
	case MT_MTU:
		req, err := msg.MTUReq()
		if err != nil {
			return err
		}
		mtu := node.MTU(ctx, req.Target)
		if err != nil {
			return err
		}
		WriteAskMessage(resFr, msg.GetRequestID(), MT_MTU, MTURes{
			MTU: mtu,
		})
	case MT_FindAddr:
		req, err := msg.FindAddrReq()
		if err != nil {
			return err
		}
		addr, err := node.FindAddr(ctx, req.Prefix, req.Nbits)
		if err != nil {
			WriteError[FindAddrRes](resFr, msg.GetRequestID(), MT_FindAddr, err)
		} else {
			WriteSuccess(resFr, msg.GetRequestID(), MT_FindAddr, FindAddrRes{
				Addr: addr,
			})
		}
	case MT_PublicKey:
		req, err := msg.LookupPublicKeyReq()
		if err != nil {
			return err
		}
		pubKey, err := node.LookupPublicKey(ctx, req.Target)
		if err != nil {
			WriteError[LookupPublicKeyRes](resFr, msg.GetRequestID(), MT_PublicKey, err)
		} else {
			WriteSuccess(resFr, msg.GetRequestID(), MT_PublicKey, LookupPublicKeyRes{
				PublicKey: inet256.MarshalPublicKey(nil, pubKey),
			})
		}
	case MT_KeepAlive:
	default:
		return fmt.Errorf("unrecognized message type %v", msg.GetType())
	}
	return nil
}
