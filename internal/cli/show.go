package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

// NewShow returns the `show` subcommand which prints the files each adapter
// would manage if `apply` were run now. Useful for auditing what harness-sync
// will write to your system before committing to it.
func NewShow(reg *adapter.Registry) *cobra.Command {
	var (
		root string
		all  bool
	)
	cmd := &cobra.Command{
		Use:   "show [harness...]",
		Short: "List files harness-sync manages for each harness",
		Long: `Renders the canonical bundle through each adapter and prints
the resulting targets without writing anything. By default, shows only detected
harnesses; pass --all to include adapters whose harness is not installed, or
positional args to select specific harnesses.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolveRoot(root)
			if err != nil {
				return err
			}
			b, err := canonical.Load(r)
			if err != nil {
				return err
			}

			var selected []adapter.Adapter
			switch {
			case len(args) > 0:
				for _, name := range args {
					a, ok := reg.Get(name)
					if !ok {
						return fmt.Errorf("unknown adapter %q", name)
					}
					selected = append(selected, a)
				}
			case all:
				selected = reg.All()
			default:
				for _, a := range reg.All() {
					if a.Detect() {
						selected = append(selected, a)
					}
				}
			}

			w := cmd.OutOrStdout()
			for _, a := range selected {
				fs, err := a.Render(b)
				if err != nil {
					if _, werr := fmt.Fprintf(w, "%s: render error: %v\n", a.Name(), err); werr != nil {
						return werr
					}
					continue
				}
				if _, werr := fmt.Fprintf(w, "%s (%s)\n", a.Name(), detectLabel(a)); werr != nil {
					return werr
				}
				var iterErr error
				fs.ForEach(func(f adapter.File) {
					if iterErr != nil {
						return
					}
					switch f.Kind {
					case adapter.SymlinkDir, adapter.SymlinkFile:
						_, iterErr = fmt.Fprintf(w, "  %-12s %s -> %s\n",
							f.Kind.String(), short(f.Dest), short(f.SymlinkTarget))
					case adapter.RenderedFile:
						_, iterErr = fmt.Fprintf(w, "  %-12s %s (%d bytes)\n",
							f.Kind.String(), short(f.Dest), len(f.Content))
					}
				})
				if iterErr != nil {
					return iterErr
				}
				if _, werr := fmt.Fprintln(w); werr != nil {
					return werr
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "canonical root (default ~/.config/harness-sync)")
	cmd.Flags().BoolVar(&all, "all", false, "include adapters whose harness is not detected")
	return cmd
}

func detectLabel(a adapter.Adapter) string {
	if a.Detect() {
		return "detected"
	}
	return "not detected"
}

// short collapses the user's home dir to "~" for cleaner output.
func short(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if len(p) > len(home) && p[:len(home)+1] == home+string(filepath.Separator) {
		return "~" + p[len(home):]
	}
	return p
}
