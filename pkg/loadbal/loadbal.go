package loadbal

import (
	"context"
	"io"
	"net"

	"github.com/inet256/inet256/pkg/inet256"
)

type StreamDialer interface {
	Dial(ctx context.Context, target inet256.Addr) (net.Conn, error)
}

type StreamFrontend interface {
	Accept(ctx context.Context) (net.Conn, error)
}

type StreamMuxSession interface {
	OpenStream(ctx context.Context) (io.ReadWriteCloser, error)
	AcceptStream(ctx context.Context) (io.ReadWriteCloser, error)
}

type StreamMuxFrontend interface {
	Accept(ctx context.Context) (StreamMuxSession, error)
}

type StreamBackend interface {
	Dial(ctx context.Context) (net.Conn, error)
}

// func ServeStreamMux(ctx context.Context, frontend StreamMuxFrontend, backends []StreamBackend) error {
// 	eg := errgroup.Group{}
// 	for {
// 		sess, err := frontend.Accept(ctx)
// 		if err != nil {
// 			return err
// 		}
// 	}
//  return eg.Wait()
// }

// func ServeStream(ctx context.Context, frontend StreamFrontend, backends []StreamBackend) error {
// 	for {
// 		conn, err := frontend.Accept(ctx)
// 		if err != nil {
// 			return err
// 		}
// 		logctx.Infof(ctx, "accepted connection", slog.Any("remote_addr", conn.RemoteAddr()))
// 		go func() {
// 			backends.
// 		}()
// 	}
// 	return nil
// }
