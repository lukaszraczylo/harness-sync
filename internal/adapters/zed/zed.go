// Package zed is the Zed editor harness adapter.
//
// Zed uses ~/.config/zed/settings.json with many user-managed keys.
// harness-sync manages: context_servers (MCP), language_models.openai, agent.default_model.
// All other keys are preserved via JSON merge.
package zed

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "zed"

// Adapter implements adapter.Adapter for the Zed harness.
type Adapter struct {
	*adapter.Base
}

// WithHome overrides the home directory used to resolve target paths.
func WithHome(h string) adapter.BaseOption { return func(b *adapter.Base) { b.Home = h } }

// New returns a new Adapter with the given options applied.
func New(opts ...adapter.BaseOption) *Adapter {
	return &Adapter{Base: adapter.NewBase(opts...)}
}

// Name returns the harness identifier.
func (a *Adapter) Name() string { return name }

// Capabilities declares what zed harness-sync manages.
// Zed does not have a built-in subscription — it routes through the configured gateway.
func (a *Adapter) Capabilities() adapter.HarnessCapabilities {
	return adapter.HarnessCapabilities{
		ManagesProviders:    true,
		ManagesModels:       true,
		ManagesMCP:          false, // Agent panel uses Extensions; context_servers key causes serde parse errors
		ManagesSkills:       false,
		ManagesRules:        true,
		ManagesInstructions: true,
		HasBuiltInSub:       false,
	}
}

// Detect returns true when ~/.config/zed/ exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.Home, ".config", "zed"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to Zed.
// settings.json is MERGED to preserve all user-managed Zed configuration.
// Manages: language_models.openai_compatible, agent.default_model.
//
// NOTE: context_servers is NOT written. Zed 1.3+ uses Extension-based MCP
// servers in the Agent panel; writing context_servers causes a serde parse
// error (ContextServerSettingsContent untagged enum bug) that aborts loading
// the entire settings.json. We clear any existing context_servers key to fix
// pre-existing parse errors from user-configured entries.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	cfgPath := filepath.Join(a.Home, ".config", "zed", "settings.json")
	existing, err := common.ReadExistingFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cfgPath, err)
	}

	// Set context_servers to nil so MergeJSONKeys removes the key entirely,
	// clearing any entries that trigger the serde ContextServerSettingsContent bug.
	overlay := map[string]any{
		"context_servers": nil,
	}

	if b.Profile.Gateway.URL != "" {
		overlay["language_models"] = zedLanguageModels(&b.Profile)
		overlay["agent"] = zedAgentBlock(existing, &b.Profile)
	} else {
		overlay["language_models"] = nil
	}

	merged, err := common.MergeJSONKeys(existing, overlay)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    cfgPath,
		Kind:    adapter.RenderedFile,
		Content: merged,
		// NoMerge: Zed settings.json is JSONC (trailing commas, comments). The
		// JSON merge above already reconciles at the key level; git 3-way merge
		// on JSONC causes spurious conflicts on whitespace/comment differences.
		NoMerge: true,
	})

	// Zed reads personal/global instructions from ~/.config/zed/AGENTS.md
	// (always-on for the Agent across all projects). Zed has no rules directory,
	// so fold rules into it.
	if instr := b.InstructionTextWithRules(name); instr != "" {
		fs.Add(adapter.File{
			Dest:    filepath.Join(a.Home, ".config", "zed", "AGENTS.md"),
			Kind:    adapter.RenderedFile,
			Content: []byte(instr),
		})
	}

	return fs, nil
}

// zedLanguageModels builds language_models.openai_compatible from the profile
// gateway. Each named entry becomes its own provider in Zed's Agent panel.
// API key is read by Zed from env var HARNESS_SYNC_GATEWAY_API_KEY at runtime.
func zedLanguageModels(p *canonical.Profile) map[string]any {
	entry := map[string]any{"api_url": p.Gateway.URL}
	if len(p.Models) > 0 {
		models := make([]map[string]any, 0, len(p.Models))
		for _, m := range p.Models {
			me := map[string]any{
				"name":       m.ID,
				"max_tokens": 200000,
				"capabilities": map[string]any{
					"tools":               true,
					"images":              false,
					"parallel_tool_calls": false,
					"prompt_cache_key":    false,
				},
			}
			if m.Alias != "" {
				me["display_name"] = m.Alias
			}
			models = append(models, me)
		}
		entry["available_models"] = models
	}
	return map[string]any{
		"openai_compatible": map[string]any{
			common.GatewayProviderKey(p.Gateway.URL): entry,
		},
	}
}

// zedAgentBlock merges default_model into the existing agent block,
// preserving all other user-managed agent settings.
func zedAgentBlock(existing []byte, p *canonical.Profile) map[string]any {
	base := map[string]any{}
	if len(existing) > 0 {
		var raw map[string]any
		clean := common.StripJSONComments(string(existing))
		if json.Unmarshal([]byte(clean), &raw) == nil {
			if ag, ok := raw["agent"].(map[string]any); ok {
				base = ag
			}
		}
	}
	if p.Gateway.DefaultModel != "" {
		base["default_model"] = map[string]any{
			"provider": common.GatewayProviderKey(p.Gateway.URL),
			"model":    p.Gateway.DefaultModel,
		}
	}
	return base
}

// Import reads context_servers from Zed settings.json back to MCP servers.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
