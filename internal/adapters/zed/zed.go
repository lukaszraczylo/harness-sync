// Package zed is the Zed editor harness adapter.
//
// Zed uses ~/.config/zed/settings.json with many user-managed keys.
// harness-sync manages: context_servers (MCP), language_models.openai, agent.default_model.
// All other keys are preserved via JSON merge.
package zed

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "zed"

// Adapter implements adapter.Adapter for the Zed harness.
type Adapter struct{ home string }

// Option configures an Adapter.
type Option func(*Adapter)

// WithHome overrides the home directory (defaults to os.UserHomeDir).
func WithHome(h string) Option { return func(a *Adapter) { a.home = h } }

// New returns a new Adapter with the given options applied.
func New(opts ...Option) *Adapter {
	a := &Adapter{home: common.DefaultHome()}
	for _, o := range opts {
		o(a)
	}
	return a
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
		ManagesInstructions: false,
		HasBuiltInSub:       false,
	}
}

// Detect returns true when ~/.config/zed/ exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".config", "zed"))
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
	cfgPath := filepath.Join(a.home, ".config", "zed", "settings.json")
	existing, _ := os.ReadFile(cfgPath)

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

	return fs, nil
}

// zedLanguageModels builds language_models.openai_compatible from the profile
// gateway. Each named entry becomes its own provider in Zed's Agent panel.
// The gateway token is written inline as api_key, matching every other
// provider adapter and the dummy/${VAR} contract (apply does not resolve
// ${VAR}; Zed/the downstream resolves it).
// NOTE: confirm api_key is the field Zed's openai_compatible schema reads; Zed
// may also accept keys via its Agent panel UI (stored in the OS keychain).
func zedLanguageModels(p *canonical.Profile) map[string]any {
	entry := map[string]any{"api_url": p.Gateway.URL}
	if p.Gateway.Token != "" {
		entry["api_key"] = p.Gateway.Token
	}
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
