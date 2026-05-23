package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/apply"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/gitx"
)

// NewApply returns the `apply` subcommand. It renders the canonical bundle
// to each detected (or selected) harness using the apply pipeline.
func NewApply(reg *adapter.Registry) *cobra.Command {
	var (
		dryRun bool
		force  bool
		root   string
	)
	cmd := &cobra.Command{
		Use:   "apply [harness...]",
		Short: "Render canonical and propagate to harnesses",
		RunE: func(cmd *cobra.Command, args []string) error {
			if root == "" {
				h, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				root = filepath.Join(h, ".config", "harness-sync")
			}
			b, err := canonical.Load(root)
			if err != nil {
				return err
			}

			var selected []adapter.Adapter
			if len(args) == 0 {
				for _, a := range reg.All() {
					if a.Detect() {
						selected = append(selected, a)
					}
				}
			} else {
				for _, name := range args {
					a, ok := reg.Get(name)
					if !ok {
						return fmt.Errorf("unknown adapter %q", name)
					}
					selected = append(selected, a)
				}
			}

			opt := apply.Options{
				Bundle:   b,
				Adapters: selected,
				Repo:     gitx.New(root),
				DryRun:   dryRun,
				Force:    force,
			}
			rep, err := apply.Run(opt)
			if err != nil {
				return err
			}
			mode := "applied"
			if dryRun {
				mode = "dry-run"
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: %d written, %d skipped, %d conflicts\n",
				mode, rep.Written, rep.Skipped, rep.Conflicts); err != nil {
				return err
			}
			for _, a := range rep.Actions {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %-10s %-12s %s %s\n", a.Adapter, a.Kind, a.Dest, a.Note); err != nil {
					return err
				}
			}
			if rep.Conflicts > 0 {
				return fmt.Errorf("%d conflicts; resolve .rej files", rep.Conflicts)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print actions without writing")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite without 3-way merge")
	cmd.Flags().StringVar(&root, "root", "", "canonical root (default ~/.config/harness-sync)")
	return cmd
}
