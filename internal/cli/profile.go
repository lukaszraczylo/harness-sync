package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

// NewProfile returns the `profile` subcommand with `list` and `use`.
// reg is used by the `use --apply` flag to invoke apply after switching.
func NewProfile(reg *adapter.Registry) *cobra.Command {
	var root string
	rootFlag := func(c *cobra.Command) {
		c.Flags().StringVar(&root, "root", "", "canonical root (default ~/.config/harness-sync)")
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, err := resolveRoot(root)
			if err != nil {
				return err
			}
			entries, err := os.ReadDir(filepath.Join(r, "profiles"))
			if err != nil {
				return err
			}
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				if n, ok := strings.CutSuffix(e.Name(), ".yaml"); ok {
					names = append(names, n)
				}
			}
			sort.Strings(names)
			for _, n := range names {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), n)
			}
			return nil
		},
	}
	rootFlag(list)

	var applyAfter bool
	use := &cobra.Command{
		Use:   "use <name>",
		Short: "Set active profile (rewrites config.yaml)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolveRoot(root)
			if err != nil {
				return err
			}
			target := args[0]
			profPath := filepath.Join(r, "profiles", target+".yaml")
			if _, err := os.Stat(profPath); err != nil {
				return fmt.Errorf("profile %q not found at %s", target, profPath)
			}
			configPath := filepath.Join(r, "config.yaml")
			existing, _ := os.ReadFile(configPath)
			updated := setActiveProfile(string(existing), target)
			if err := os.WriteFile(configPath, []byte(updated), 0o600); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(),
				"Switched active profile to %q. Run `harness-sync apply` to propagate.\n", target)
			// Issue 4: --apply flag auto-invokes apply after switching.
			if applyAfter {
				applyCmd := NewApply(reg)
				applyCmd.SetOut(cmd.OutOrStdout())
				applyCmd.SetErr(cmd.ErrOrStderr())
				applyCmd.SetArgs([]string{"--root", r, "--yes"})
				return applyCmd.Execute()
			}
			return nil
		},
	}
	rootFlag(use)
	use.Flags().BoolVar(&applyAfter, "apply", false, "run apply automatically after switching profile")

	rename := &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a profile (file + name field + active_profile if needed)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := resolveRoot(root)
			if err != nil {
				return err
			}
			oldName, newName := args[0], args[1]
			oldPath := filepath.Join(r, "profiles", oldName+".yaml")
			newPath := filepath.Join(r, "profiles", newName+".yaml")

			if _, statErr := os.Stat(oldPath); statErr != nil {
				return fmt.Errorf("profile %q not found", oldName)
			}
			if _, statErr := os.Stat(newPath); statErr == nil {
				return fmt.Errorf("profile %q already exists", newName)
			}

			body, err := os.ReadFile(oldPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(newPath, []byte(setProfileName(string(body), newName)), 0o600); err != nil {
				return err
			}
			if err := os.Remove(oldPath); err != nil {
				_ = os.Remove(newPath)
				return err
			}

			// Update active_profile in config.yaml if it points at the old name.
			configPath := filepath.Join(r, "config.yaml")
			if existing, readErr := os.ReadFile(configPath); readErr == nil {
				if isActiveProfile(string(existing), oldName) {
					_ = os.WriteFile(configPath, []byte(setActiveProfile(string(existing), newName)), 0o600)
				}
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Renamed profile %q → %q.\n", oldName, newName)
			return nil
		},
	}
	rootFlag(rename)

	cmd := &cobra.Command{Use: "profile", Short: "Manage profiles"}
	cmd.AddCommand(list, use, rename)
	return cmd
}

func setActiveProfile(existing, name string) string {
	lines := strings.Split(existing, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "active_profile:") {
			lines[i] = "active_profile: " + name
			return strings.Join(lines, "\n")
		}
	}
	return strings.TrimRight(existing, "\n") + "\nactive_profile: " + name + "\n"
}

func isActiveProfile(configYAML, name string) bool {
	for l := range strings.SplitSeq(configYAML, "\n") {
		if rest, ok := strings.CutPrefix(l, "active_profile:"); ok {
			return strings.TrimSpace(rest) == name
		}
	}
	return false
}

func setProfileName(body, name string) string {
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "name:") {
			lines[i] = "name: " + name
			return strings.Join(lines, "\n")
		}
	}
	return "name: " + name + "\n" + strings.TrimLeft(body, "\n")
}
