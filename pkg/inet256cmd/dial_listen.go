package inet256cmd

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256lb"
)

func NewDialCmd(newNode func(ctx context.Context, privateKey inet256.PrivateKey) (inet256.Node, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "dial",
		Short: "dials a new connection using a transport protocol on INET256",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dst, err := inet256.ParseAddrBase64([]byte(args[0]))
			if err != nil {
				return err
			}
			var privateKey inet256.PrivateKey // TODO: flag
			var protocol string               // TODO: flag

			var dialFunc func(node inet256.Node) (net.Conn, error)
			switch protocol {
			case "utp":
				dialFunc = func(node inet256.Node) (net.Conn, error) {
					return nil, nil
				}
			default:
				return fmt.Errorf("protocol unknown %q", protocol)
			}
			node, err := newNode(ctx, privateKey)
			if err != nil {
				return nil
			}
			defer node.Close()
			conn, err := dialFunc(node)
			if err != nil {
				return err
			}
			defer conn.Close()
			pair := rwPair{
				Reader: cmd.InOrStdIn(),
				Writer: cmd.OutOrStdout(),
			}
			return inet256lb.PlumbRWC(pair, conn)
		},
	}
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
	return &cobra.Command{
		Use:   "listen",
		Short: "listens for new connections using a transport protocol on INET256",
		RunE: func(cmd *cobra.Command, args []string) error {
			var privateKey inet256.PrivateKey // TODO: flag
			var protocol string               // TODO: flag

			if len(args) < 0 {
				return errors.New("no endpoints")
			}
			var endpoints []inet256lb.StreamEndpoint
			for i := range args {
				e, err := inet256lb.MakeStreamEndpoint(args[i])
				if err != nil {
					return err
				}
				endpoints = append(endpoints, e)
			}

			var listenFunc func(node inet256.Node) (net.Listener, error) {

			}
			node, err := newNode(ctx, privateKey)
			if err != nil {
				return nil
			}
			defer node.Close()	
			defer conn.Close()
			pair := rwPair{
				Reader: cmd.InOrStdIn(),
				Writer: cmd.OutOrStdout(),
			}
			return inet256lb.PlumbRWC(pair, conn)
		},
	}
}
