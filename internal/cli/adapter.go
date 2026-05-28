package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

// capabilityTags returns the enabled capability flags of an adapter as a
// stable, machine-readable list (e.g. "providers,models,mcp").
func capabilityTags(c adapter.HarnessCapabilities) []string {
	var tags []string
	if c.ManagesProviders {
		tags = append(tags, "providers")
	}
	if c.ManagesModels {
		tags = append(tags, "models")
	}
	if c.ManagesMCP {
		tags = append(tags, "mcp")
	}
	if c.ManagesSkills {
		tags = append(tags, "skills")
	}
	if c.ManagesInstructions {
		tags = append(tags, "instructions")
	}
	if c.HasBuiltInSub {
		tags = append(tags, "built-in-subscription")
	}
	return tags
}

// NewAdapter returns the `adapter` parent command with a `list` subcommand.
func NewAdapter(reg *adapter.Registry) *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List registered adapters and what each manages",
		RunE: func(cmd *cobra.Command, _ []string) error {
			for _, a := range reg.All() {
				detected := ""
				if a.Detect() {
					detected = "  [detected]"
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%-12s %s%s\n",
					a.Name(), strings.Join(capabilityTags(a.Capabilities()), ","), detected); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd := &cobra.Command{Use: "adapter", Short: "Adapter introspection"}
	cmd.AddCommand(list)
	return cmd
}
