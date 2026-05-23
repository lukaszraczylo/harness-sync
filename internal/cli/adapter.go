package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

// NewAdapter returns the `adapter` parent command with a `list` subcommand.
func NewAdapter(reg *adapter.Registry) *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List registered adapters",
		RunE: func(cmd *cobra.Command, _ []string) error {
			for _, a := range reg.All() {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), a.Name()); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd := &cobra.Command{Use: "adapter", Short: "Adapter introspection"}
	cmd.AddCommand(list)
	return cmd
}
