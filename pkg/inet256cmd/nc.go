package inet256cmd

import (
	"bufio"

	"github.com/brendoncarroll/stdctx/logctx"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func NewNetCatCmd(newNode NodeFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "nc <target_addr>",
		Short: "nc is like netcat",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			remote, err := parseAddr(args[0])
			if err != nil {
				return err
			}
			in := cmd.InOrStdin()
			out := cmd.OutOrStdout()
			pk := generateKey()
			node, err := newNode(ctx, pk)
			if err != nil {
				return err
			}
			defer node.Close()
			logctx.Infoln(ctx, node.LocalAddr())
			eg := errgroup.Group{}
			eg.Go(func() error {
				var msg inet256.Message
				for {
					if err := inet256.Receive(ctx, node, &msg); err != nil {
						return err
					}
					if msg.Src != remote {
						logctx.Warnf(ctx, "discarding message from %v", msg.Src)
						continue
					}
					out.Write(msg.Payload)
					out.Write([]byte("\n"))
				}
			})
			eg.Go(func() error {
				scn := bufio.NewScanner(in)
				for scn.Scan() {
					if err := node.Send(ctx, remote, scn.Bytes()); err != nil {
						return err
					}
				}
				return scn.Err()
			})
			return eg.Wait()
		},
	}
}

func parseAddr(x string) (inet256.Addr, error) {
	return inet256.ParseAddrBase64([]byte(x))
}
