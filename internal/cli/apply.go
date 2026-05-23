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

// isTerminal returns true when os.Stdin is an interactive terminal.
// Used to decide whether to show the first-run prompt.
func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// NewApply returns the `apply` subcommand. It renders the canonical bundle
// to each detected (or selected) harness using the apply pipeline.
func NewApply(reg *adapter.Registry) *cobra.Command {
	var (
		dryRun          bool
		force           bool
		root            string
		allowIncomplete bool
		yes             bool
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

			// Issue 3: refuse apply when gateway is incomplete.
			if !allowIncomplete && (b.Profile.Gateway.URL == "" || b.Profile.Gateway.DefaultModel == "") {
				return fmt.Errorf("profile %q is incomplete (gateway.url or gateway.default_model is empty); edit %s/profiles/%s.yaml then re-run",
					b.Profile.Name, root, b.Profile.Name)
			}

			// Issue 5: first-time apply warning (state/ doesn't exist yet).
			if !dryRun {
				statePath := filepath.Join(root, "state")
				if _, statErr := os.Stat(statePath); os.IsNotExist(statErr) {
					if !yes && isTerminal() {
						_, _ = fmt.Fprintf(cmd.OutOrStdout(),
							"First-time apply to detected harnesses. harness-sync will:\n"+
								"  - move existing target files to %s/backups/<harness>/\n"+
								"  - replace them with symlinks to %s/skills, %s/agents\n"+
								"  - render LLM configs from the active profile\n\n"+
								"Continue? [y/N] ", root, root, root)
						var answer string
						if _, scanErr := fmt.Fscan(cmd.InOrStdin(), &answer); scanErr != nil || (answer != "y" && answer != "Y") {
							return fmt.Errorf("aborted")
						}
					}
				}
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
	cmd.Flags().BoolVar(&allowIncomplete, "allow-incomplete", false, "apply even when gateway.url or gateway.default_model is empty (for testing)")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip first-run confirmation prompt")
	return cmd
}
