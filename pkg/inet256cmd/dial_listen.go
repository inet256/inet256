package inet256cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256lb"
)

func NewDialCmd(newNode func(ctx context.Context, privateKey inet256.PrivateKey) (inet256.Node, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dial",
		Short: "dials a new connection using a transport protocol on INET256",
		Args:  cobra.ExactArgs(1),
	}
	protocol := flagProtocol(cmd)
	privateKeyPath := flagPrivateKey(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		dst, err := inet256.ParseAddrBase64([]byte(args[0]))
		if err != nil {
			return err
		}
		if *privateKeyPath == "" {
			return errors.New("must provide private-key")
		}
		privateKey, err := loadPrivateKeyFromFile(*privateKeyPath)
		if err != nil {
			return err
		}

		var newEndpoint func(inet256.Node) inet256lb.StreamEndpoint
		switch *protocol {
		case "utp":
			newEndpoint = func(node inet256.Node) inet256lb.StreamEndpoint {
				return inet256lb.NewUTPBackend(node, dst)
			}
		default:
			return fmt.Errorf("protocol unknown %q", *protocol)
		}

		node, err := newNode(ctx, privateKey)
		if err != nil {
			return nil
		}
		defer node.Close()
		endpoint := newEndpoint(node)
		defer endpoint.Close()

		conn, err := endpoint.Open(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()
		pair := rwPair{
			Reader: cmd.InOrStdin(),
			Writer: cmd.OutOrStdout(),
		}
		return inet256lb.PlumbRWC(pair, conn)
	}
	return cmd
}

type rwPair struct {
	io.Reader
	io.Writer
}

func (p rwPair) Close() error {
	if c, ok := p.Reader.(io.Closer); ok {
		c.Close()
	}
	if c, ok := p.Writer.(io.Closer); ok {
		c.Close()
	}
	return nil
}

func NewListenCmd(newNode func(ctx context.Context, privateKey inet256.PrivateKey) (inet256.Node, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "listens for new connections using a transport protocol on INET256",
		Args:  cobra.MinimumNArgs(1),
	}
	protocol := flagProtocol(cmd)
	privateKeyPath := flagPrivateKey(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *privateKeyPath == "" {
			return errors.New("must provide private-key")
		}
		privateKey, err := loadPrivateKeyFromFile(*privateKeyPath)
		if err != nil {
			return err
		}

		node, err := newNode(ctx, privateKey)
		if err != nil {
			return nil
		}
		bal := inet256lb.NewStreamBalancer()

		// backends
		if len(args) == 1 && args[0] == "stdio" {
			bal.AddBackend("stdio", newRWBackend(cmd.InOrStdin(), cmd.OutOrStdout()))
		} else {
			for i := range args {
				e, err := inet256lb.MakeStreamBackend(args[i], func() inet256.Node { return node })
				if err != nil {
					return err
				}
				bal.AddBackend(args[i], e)
			}
		}

		// frontend
		frontend, err := inet256lb.MakeStreamFrontend(*protocol, node)
		if err != nil {
			return err
		}
		return bal.ServeFrontend(ctx, frontend)
	}
	return cmd
}

type rwBackend struct {
	r io.Reader
	w io.Writer

	closeOnce sync.Once
	sem       chan struct{}
	closed    chan struct{}
}

func newRWBackend(r io.Reader, w io.Writer) *rwBackend {
	return &rwBackend{
		sem:    make(chan struct{}, 1),
		closed: make(chan struct{}),
	}
}

func (b *rwBackend) Open(ctx context.Context) (net.Conn, error) {
	select {
	case <-b.closed:
		return nil, net.ErrClosed
	default:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.closed:
		return nil, net.ErrClosed
	case b.sem <- struct{}{}:
		return &rwConn{
			r: b.r,
			w: b.w,
			close: func() error {
				<-b.sem
				return nil
			},
		}, nil
	}
}

func (b *rwBackend) Close() error {
	b.closeOnce.Do(func() {
		close(b.closed)
	})
	return nil
}

type rwConn struct {
	r     io.Reader
	w     io.Writer
	close func() error
}

func (c *rwConn) Write(buf []byte) (int, error) {
	return c.w.Write(buf)
}

func (c *rwConn) Read(buf []byte) (int, error) {
	return c.r.Read(buf)
}

func (c *rwConn) LocalAddr() net.Addr {
	return rwAddr{}
}

func (c *rwConn) RemoteAddr() net.Addr {
	return rwAddr{}
}

func (c *rwConn) SetDeadline(d time.Time) error {
	return nil
}

func (c *rwConn) SetReadDeadline(d time.Time) error {
	return nil
}

func (c *rwConn) SetWriteDeadline(d time.Time) error {
	return nil
}

func (c *rwConn) Close() error {
	return c.close()
}

type rwAddr struct {
}

func (rwAddr) Network() string {
	return "stdio"
}

func (rwAddr) String() string {
	return "stdio"
}

func flagPrivateKey(cmd *cobra.Command) *string {
	return cmd.Flags().StringP("private-key", "k", "", "--private-key=./path/to/private/key.pem")
}

func flagProtocol(cmd *cobra.Command) *string {
	return cmd.Flags().StringP("protocol", "p", "", "--protocol=utp")
}
