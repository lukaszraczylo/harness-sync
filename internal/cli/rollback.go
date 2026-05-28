package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/apply"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/gitx"
)

// NewRollback returns the `rollback` subcommand. It reverts the last N apply
// commits in the canonical repo with `git revert`, then re-applies so the
// reverted canonical state is propagated back out to the harness target files
// (a bare revert only moves the in-repo state/ snapshots).
func NewRollback(reg *adapter.Registry) *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "rollback [n]",
		Short: "Revert the last N apply commits and re-apply to the harnesses",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolveRoot(root)
			if err != nil {
				return err
			}
			n := 1
			if len(args) == 1 {
				v, convErr := strconv.Atoi(args[0])
				if convErr != nil || v < 1 {
					return fmt.Errorf("rollback: n must be a positive integer, got %q", args[0])
				}
				n = v
			}

			repo := gitx.New(r)
			if err = repo.Revert(n); err != nil {
				return err
			}

			// Propagate the reverted canonical state outward to the harnesses.
			b, err := canonical.Load(r)
			if err != nil {
				return err
			}
			selected, err := selectAdapters(reg, nil, b.Config.EnabledHarnesses)
			if err != nil {
				return err
			}
			// Force: a plain re-apply would 3-way-merge the reverted render (now
			// equal to the reverted state base) against the current harness files
			// and keep the current content. Rollback must overwrite the harness
			// files with the reverted state.
			rep, err := apply.Run(apply.Options{Bundle: b, Adapters: selected, Repo: repo, Force: true})
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(),
				"rolled back %d commit(s); re-applied: %d written, %d skipped, %d conflicts\n",
				n, rep.Written, rep.Skipped, rep.Conflicts); err != nil {
				return err
			}
			if rep.Conflicts > 0 {
				return fmt.Errorf("%d conflicts after rollback; resolve .rej files", rep.Conflicts)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "canonical root")
	return cmd
}
