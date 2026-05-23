package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/gitx"
)

// NewRollback returns the `rollback` subcommand. It reverts the last N apply
// commits in the canonical repo by calling `git revert` over a range.
func NewRollback() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "rollback [n]",
		Short: "Revert the last N apply commits in the canonical repo",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolveRoot(root)
			if err != nil {
				return err
			}
			n := 1
			if len(args) == 1 {
				v, err := strconv.Atoi(args[0])
				if err != nil || v < 1 {
					return cmd.Help()
				}
				n = v
			}
			return gitx.New(r).Revert(n)
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "canonical root")
	return cmd
}
