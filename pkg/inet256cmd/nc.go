package inet256cmd

import (
	"bufio"
	"context"
	"encoding/base64"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	rootCmd.AddCommand(ncCmd)
}

var ncCmd = &cobra.Command{
	Use:   "nc",
	Short: "nc is like netcat",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("must provide host")
		}
		remote, err := parseAddr(args[0])
		if err != nil {
			return err
		}
		ctx := context.Background()
		in := cmd.InOrStdin()
		out := cmd.OutOrStdout()
		pk := generateKey()
		node, err := newNode(ctx, pk)
		if err != nil {
			return err
		}
		defer node.Close()
		logrus.Info(node.LocalAddr())
		eg := errgroup.Group{}
		eg.Go(func() error {
			buf := make([]byte, inet256.TransportMTU)
			var src, dst inet256.Addr
			for {
				n, err := node.Recv(ctx, &src, &dst, buf)
				if err != nil {
					return err
				}
				if src != remote {
					logrus.Warnf("discarding message from %v", src)
					continue
				}
				data := buf[:n]
				out.Write(data)
				out.Write([]byte("\n"))
			}
		})
		eg.Go(func() error {
			scn := bufio.NewScanner(in)
			for scn.Scan() {
				if err := node.Tell(ctx, remote, scn.Bytes()); err != nil {
					return err
				}
			}
			return scn.Err()
		})
		return eg.Wait()
	},
}

func parseAddr(x string) (inet256.Addr, error) {
	data, err := base64.RawURLEncoding.DecodeString(x)
	if err != nil {
		return inet256.Addr{}, err
	}
	return inet256.AddrFromBytes(data), nil
}
