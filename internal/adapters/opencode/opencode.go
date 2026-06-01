// Package opencode is the opencode harness adapter.
package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "opencode"

// Adapter implements adapter.Adapter for the opencode harness.
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

// Capabilities declares what opencode harness-sync manages.
func (a *Adapter) Capabilities() adapter.HarnessCapabilities {
	return adapter.HarnessCapabilities{
		ManagesProviders:    true,
		ManagesModels:       true,
		ManagesMCP:          true,
		ManagesSkills:       true,
		ManagesInstructions: true,
		HasBuiltInSub:       false,
	}
}

// Detect returns true when ~/.config/opencode exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.Home, ".config", "opencode"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to opencode.
// opencode.jsonc is MERGED to preserve user-managed keys ($schema, agent,
// instructions, …). Correct keys: "provider" (singular, map), "model" (string),
// "mcp" (map with type: local|remote entries).
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.Home, ".config", "opencode")

	// opencode reads skills from ~/.config/opencode/skills/<name>/SKILL.md
	fs.Add(adapter.File{
		Dest:          filepath.Join(base, "skills"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "skills"),
	})

	cfgPath := filepath.Join(base, "opencode.jsonc")
	existing, err := common.ReadExistingFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cfgPath, err)
	}

	// Build the provider map, then absorb any existing providers at the same
	// gateway URL so duplicates don't accumulate across apply runs.
	provMap := common.ProvidersAsMap(&b.Profile)
	if len(existing) > 0 {
		var base map[string]any
		clean := common.StripJSONComments(string(existing))
		if json.Unmarshal([]byte(clean), &base) == nil {
			if existingProv, ok := base["provider"].(map[string]any); ok {
				for k, v := range existingProv {
					if _, ours := provMap[k]; !ours {
						provMap[k] = v
					}
				}
				provMap = common.AbsorbDuplicateProviders(provMap, common.GatewayProviderKey(b.Profile.Gateway.URL), b.Profile.Gateway.URL)
			}
		}
	}

	overlay := map[string]any{
		"provider": provMap,
		"model":    common.KiloModelString(&b.Profile),
	}
	if mcp := common.BuildMCPMapStyled(&b.MCP, common.MCPOpencodeStyle); len(mcp) > 0 {
		overlay["mcp"] = mcp
	}
	merged, err := common.MergeJSONKeys(existing, overlay)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    cfgPath,
		Kind:    adapter.RenderedFile,
		Content: merged,
	})

	instructions := b.Instructions.Global
	if override, ok := b.Instructions.PerHarness[name]; ok && override != "" {
		instructions = override
	}
	fs.Add(adapter.File{
		Dest:    filepath.Join(base, "AGENTS.md"),
		Kind:    adapter.RenderedFile,
		Content: []byte(instructions),
	})

	return fs, nil
}

// Import reads opencode config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
