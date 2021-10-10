package inet256cmd

import (
	"fmt"

	"github.com/brendoncarroll/go-p2p/d/celltracker"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createTrackerCmd)
}

var createTrackerCmd = &cobra.Command{
	Use:    "create-tracker-token",
	Short:  "creates a new tracker config",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("must provide endpoint")
		}
		endpoint := args[0]
		token := celltracker.GenerateToken(endpoint)
		w := cmd.OutOrStdout()
		fmt.Fprintln(w, token)
		return nil
	},
}
