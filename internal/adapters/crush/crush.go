// Package crush is the charmbracelet/crush harness adapter.
package crush

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "crush"

// Adapter implements adapter.Adapter for the crush harness.
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

// Capabilities declares what crush harness-sync manages.
func (a *Adapter) Capabilities() adapter.HarnessCapabilities {
	return adapter.HarnessCapabilities{
		ManagesProviders:    true,
		ManagesModels:       true,
		ManagesMCP:          true,
		ManagesSkills:       true,
		ManagesRules:        true,
		ManagesInstructions: true,
		HasBuiltInSub:       false,
	}
}

// Detect returns true when ~/.config/crush exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.Home, ".config", "crush"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to crush.
// crush.json is MERGED so user-managed keys ($schema, lsp, options, permissions) are preserved.
// MCP key is "mcp" (not "mcpServers"); entries include a "type" field.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.Home, ".config", "crush")

	fs.Add(adapter.File{
		Dest:          filepath.Join(base, "skills"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "skills"),
	})

	// crush has no global instructions file and no rules directory: it loads
	// context from the files listed in options.context_paths (defaults are always
	// prepended; absolute paths load verbatim; the global config applies to every
	// project). Write a global AGENTS.md with rules folded in, and register its
	// absolute path so crush loads it in every session.
	agentsMD := filepath.Join(base, "AGENTS.md")
	fs.Add(adapter.File{
		Dest:    agentsMD,
		Kind:    adapter.RenderedFile,
		Content: []byte(b.InstructionTextWithRules(name)),
	})

	cfgPath := filepath.Join(base, "crush.json")
	existing, err := common.ReadExistingFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cfgPath, err)
	}
	overlay := map[string]any{}
	overlay["options"] = crushOptionsWithContextPath(existing, agentsMD)
	if providers := common.ProvidersAsCrushMap(&b.Profile); len(providers) > 0 {
		overlay["providers"] = providers
	}
	if roleModels := common.CrushRoleModels(&b.Profile); len(roleModels) > 0 {
		overlay["models"] = roleModels
	}
	overlay["default_model"] = b.Profile.Gateway.DefaultModel
	if mcp := common.BuildMCPMapStyled(&b.MCP, common.MCPCrushStyle); len(mcp) > 0 {
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

	return fs, nil
}

// crushOptionsWithContextPath returns the crush "options" object with contextFile
// added to options.context_paths, preserving every other existing option key
// (lsp, permissions, …). crush always prepends its built-in default context
// paths at load time, so only the extra path is set here. Idempotent.
func crushOptionsWithContextPath(existing []byte, contextFile string) map[string]any {
	opts := map[string]any{}
	if len(existing) > 0 {
		var raw map[string]any
		clean := common.StripJSONComments(string(existing))
		if json.Unmarshal([]byte(clean), &raw) == nil {
			if o, ok := raw["options"].(map[string]any); ok {
				opts = o
			}
		}
	}
	var paths []any
	if existingPaths, ok := opts["context_paths"].([]any); ok {
		paths = existingPaths
	}
	for _, p := range paths {
		if s, ok := p.(string); ok && s == contextFile {
			return opts // already registered
		}
	}
	opts["context_paths"] = append(paths, contextFile)
	return opts
}

// Import reads crush config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
