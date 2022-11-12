package inet256ipc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/inet256/inet256/pkg/inet256"
	"golang.org/x/sync/errgroup"
)

// ServeNode reads and writes packets from a Framer, using the node to
// send network data and answer requests.
// If the context is cancelled ServeNode returns nil.
func ServeNode(ctx context.Context, node inet256.Node, framer Framer) error {
	eg := errgroup.Group{}
	pool := newFramePool()
	// read loop
	eg.Go(func() error {
		for {
			fr := pool.Acquire()
			if err := framer.ReadFrame(ctx, fr); err != nil {
				if errors.Is(err, io.EOF) {
					err = nil
				}
				return err
			}
			// handle message
			if err := func() error {
				msg, err := AsMessage(fr.Body(), true)
				if err != nil {
					return err
				}
				if msg.IsTell() {
					dm := msg.DataMsg()
					if err := node.Send(ctx, dm.Addr, dm.Payload); err != nil {
						return err
					}
					pool.Release(fr)
				} else {
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
	err := eg.Wait()
	if errors.Is(err, ctx.Err()) {
		err = nil
	}
	return err
}

func handleAsk(ctx context.Context, node inet256.Node, msg Message, resFr Frame) error {
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
				PublicKey: inet256.MarshalPublicKey(pubKey),
			})
		}
	default:
		return fmt.Errorf("unrecognized message type %v", msg.GetType())
	}
	return nil
}
