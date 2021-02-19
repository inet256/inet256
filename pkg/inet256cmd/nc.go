package inet256cmd

import (
	"bufio"
	"context"
	"encoding/base64"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
		n, err := newClient(pk)
		if err != nil {
			return err
		}
		defer n.Close()
		logrus.Info(n.LocalAddr())
		n.OnRecv(func(src, _ inet256.Addr, data []byte) {
			if src != remote {
				logrus.Warnf("discarding message from %v", src)
				return
			}
			out.Write(data)
		})
		scn := bufio.NewScanner(in)
		for scn.Scan() {
			if err := n.Tell(ctx, remote, scn.Bytes()); err != nil {
				return err
			}
		}
		return scn.Err()
	},
}

func parseAddr(x string) (inet256.Addr, error) {
	data, err := base64.RawURLEncoding.DecodeString(x)
	if err != nil {
		return inet256.Addr{}, err
	}
	return inet256.AddrFromBytes(data), nil
}
