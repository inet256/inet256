package inet256sock

import (
	"context"
	"log"
	"net"

	"github.com/inet256/inet256/pkg/inet256"
	"golang.org/x/sync/errgroup"
)

// ServeNode reads and writes packets from a DGRAM unix conn, using the node to
// send network data and answer requests.
func ServeNode(ctx context.Context, node inet256.Node, pconn *net.UnixConn) error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		defer pconn.Close()
		<-ctx.Done()
		return ctx.Err()
	})
	eg.Go(func() error {
		buf := make([]byte, MaxMsgLen)
		for {
			n, _, err := pconn.ReadFromUnix(buf)
			if err != nil {
				return err
			}
			// TODO: handle messages
			log.Println(buf[:n])
		}
	})
	eg.Go(func() error {
		buf := make([]byte, MaxMsgLen)
		for {
			var sendErr error
			if err := node.Receive(ctx, func(msg inet256.Message) {
				m := MakeDataMessage(buf[:0], msg.Src, msg.Payload)
				_, err := pconn.Write(m)
				if err != nil {
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
	return eg.Wait()
}
