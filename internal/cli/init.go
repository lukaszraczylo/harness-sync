package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/gitx"
	"github.com/lukaszraczylo/harness-sync/internal/render"
	"github.com/lukaszraczylo/harness-sync/internal/ui"
)

// NewInit returns the `init` subcommand: scan for detected harnesses, import
// their content into a canonical tree, and produce an initial git commit.
func NewInit(reg *adapter.Registry) *cobra.Command {
	var (
		root     string
		home     string
		from     []string
		noPrompt bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise canonical config from existing harnesses",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			r, err := resolveRoot(root)
			if err != nil {
				return err
			}
			if home == "" {
				home, err = os.UserHomeDir()
				if err != nil {
					return err
				}
			}
			if err = os.MkdirAll(r, 0o750); err != nil {
				return err
			}
			candidates := reg.DetectedNames()
			if len(candidates) == 0 {
				return fmt.Errorf("no harnesses detected under %s", home)
			}
			var picked []string
			switch {
			case len(from) > 0:
				picked = from
			case noPrompt:
				picked = candidates
			default:
				picked, err = ui.MultiSelect("Import from which harnesses?", candidates)
				if err != nil {
					return err
				}
			}
			return runImport(cmd, reg, r, home, picked)
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "canonical root (default ~/.config/harness-sync)")
	cmd.Flags().StringVar(&home, "home", "", "home dir (default $HOME)")
	cmd.Flags().StringSliceVar(&from, "from", nil, "import from specific adapter(s), skip prompt")
	cmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "import from all detected adapters without prompting")
	return cmd
}

func runImport(cmd *cobra.Command, reg *adapter.Registry, root, home string, picked []string) error {
	bundle := &canonical.Bundle{
		Root:   root,
		Config: canonical.Config{ActiveProfile: "imported"},
	}
	seenSkills := map[string]bool{}
	seenAgents := map[string]bool{}
	seenMCP := map[string]bool{}
	var instructions []string

	for _, name := range picked {
		ad, ok := reg.Get(name)
		if !ok {
			return fmt.Errorf("unknown adapter %q", name)
		}
		res, err := ad.Import(home)
		if err != nil {
			return fmt.Errorf("import %s: %w", name, err)
		}
		for _, s := range res.Skills {
			key := s.Name + "|" + s.Body
			if seenSkills[key] {
				continue
			}
			seenSkills[key] = true
			bundle.Skills = append(bundle.Skills, s)
		}
		for _, a := range res.Agents {
			key := a.Name + "|" + a.Body
			if seenAgents[key] {
				continue
			}
			seenAgents[key] = true
			bundle.Agents = append(bundle.Agents, a)
		}
		for _, m := range res.MCP {
			if seenMCP[m.Name] {
				continue
			}
			seenMCP[m.Name] = true
			bundle.MCP.Servers = append(bundle.MCP.Servers, m)
		}
		if res.Instructions != "" {
			instructions = append(instructions, fmt.Sprintf("<!-- from %s -->\n%s", name, res.Instructions))
		}
	}
	bundle.Instructions.Global = strings.Join(instructions, "\n\n")

	if err := writeCanonical(root, bundle, picked); err != nil {
		return err
	}
	repo := gitx.New(root)
	if !repo.IsRepo() {
		if err := repo.Init(); err != nil {
			return err
		}
		if err := repo.Configure("harness-sync", "harness-sync@localhost"); err != nil {
			return err
		}
	}
	if err := repo.AddAll(); err != nil {
		return err
	}
	if err := repo.Commit(fmt.Sprintf("import from %s", strings.Join(picked, ", "))); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "imported from %v into %s\n", picked, root)
	return nil
}

func writeCanonical(root string, b *canonical.Bundle, picked []string) error {
	for _, sub := range []string{"profiles", "skills", "agents", "instructions"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o750); err != nil {
			return err
		}
	}

	cfgBody, err := render.YAML(map[string]any{
		"enabled_harnesses": picked,
		"active_profile":    "imported",
	})
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(root, "config.yaml"), cfgBody, 0o600); err != nil {
		return err
	}

	profBody, err := render.YAML(canonical.Profile{
		Name: "imported",
		Gateway: canonical.Gateway{
			URL:          "https://gateway.local",
			Token:        "dummy",
			DefaultModel: "claude-sonnet-4-6",
		},
		Models: []canonical.Model{{ID: "claude-sonnet-4-6", Alias: "sonnet"}},
	})
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(root, "profiles", "imported.yaml"), profBody, 0o600); err != nil {
		return err
	}

	if len(b.MCP.Servers) > 0 {
		mcpBody, err := render.YAML(b.MCP)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(root, "mcp.yaml"), mcpBody, 0o600); err != nil {
			return err
		}
	}

	for _, s := range b.Skills {
		path := filepath.Join(root, "skills", s.Name, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(s.Body), 0o600); err != nil {
			return err
		}
	}
	for _, a := range b.Agents {
		path := filepath.Join(root, "agents", a.Name+".md")
		if err := os.WriteFile(path, []byte(a.Body), 0o600); err != nil {
			return err
		}
	}
	if b.Instructions.Global != "" {
		if err := os.WriteFile(filepath.Join(root, "instructions", "global.md"),
			[]byte(b.Instructions.Global), 0o600); err != nil {
			return err
		}
	}
	return nil
}
