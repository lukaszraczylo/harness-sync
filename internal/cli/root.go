// Package cli provides the cobra command tree for harness-sync.
package cli

import "github.com/spf13/cobra"

func NewRoot(version string) *cobra.Command {
	return &cobra.Command{
		Use:          "harness-sync",
		Short:        "Sync skills, agents, MCP, and LLM endpoints across LLM harnesses",
		Version:      version,
		SilenceUsage: true,
	}
}
