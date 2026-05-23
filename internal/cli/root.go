// Package cli provides the cobra command tree for harness-sync.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewRoot(version string) *cobra.Command {
	return &cobra.Command{
		Use:          "harness-sync",
		Short:        "Sync skills, agents, MCP, and LLM endpoints across LLM harnesses",
		Version:      version,
		SilenceUsage: true,
	}
}

// resolveRoot returns the canonical root: the flag value when non-empty,
// else $HOME/.config/harness-sync.
func resolveRoot(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".config", "harness-sync"), nil
}
