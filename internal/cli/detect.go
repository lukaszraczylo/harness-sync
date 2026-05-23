package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

// NewDetect returns the `detect` subcommand. It prints each registered
// adapter with whether the underlying harness is installed on this machine.
func NewDetect(reg *adapter.Registry) *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "List adapters and whether each harness is present on this machine",
		RunE: func(cmd *cobra.Command, _ []string) error {
			for _, a := range reg.All() {
				status := "not detected"
				if a.Detect() {
					status = "detected"
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%-14s %s\n", a.Name(), status); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
