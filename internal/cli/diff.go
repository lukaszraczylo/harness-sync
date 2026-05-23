package cli

import (
	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

// NewDiff returns the `diff` subcommand. It is a shorthand for
// `apply --dry-run` and shows pending changes without writing.
func NewDiff(reg *adapter.Registry) *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "diff [harness...]",
		Short: "Show pending changes (alias for `apply --dry-run`)",
		RunE: func(cmd *cobra.Command, args []string) error {
			apply := NewApply(reg)
			apply.SetOut(cmd.OutOrStdout())
			apply.SetErr(cmd.ErrOrStderr())
			applyArgs := []string{"--dry-run"}
			if root != "" {
				applyArgs = append(applyArgs, "--root", root)
			}
			applyArgs = append(applyArgs, args...)
			apply.SetArgs(applyArgs)
			return apply.Execute()
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "canonical root (default ~/.config/harness-sync)")
	return cmd
}
