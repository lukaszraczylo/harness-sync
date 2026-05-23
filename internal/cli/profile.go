package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// NewProfile returns the `profile` subcommand with `list` and `use`.
func NewProfile() *cobra.Command {
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
				if strings.HasSuffix(e.Name(), ".yaml") {
					names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
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
			return os.WriteFile(configPath, []byte(updated), 0o600)
		},
	}
	rootFlag(use)

	cmd := &cobra.Command{Use: "profile", Short: "Manage profiles"}
	cmd.AddCommand(list, use)
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
