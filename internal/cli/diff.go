package cli

import (
	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

// NewDiff returns the `diff` subcommand. It is a shorthand for
// `apply --dry-run` and shows pending changes without writing.
func NewDiff(reg *adapter.Registry) *cobra.Command {
	return &cobra.Command{
		Use:   "diff [harness...]",
		Short: "Show pending changes (alias for `apply --dry-run`)",
		RunE: func(cmd *cobra.Command, args []string) error {
			apply := NewApply(reg)
			apply.SetOut(cmd.OutOrStdout())
			apply.SetErr(cmd.ErrOrStderr())
			apply.SetArgs(append([]string{"--dry-run"}, args...))
			return apply.Execute()
		},
	}
}
