// Package crush is the charmbracelet/crush harness adapter.
package crush

import (
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
		ManagesInstructions: false,
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

	cfgPath := filepath.Join(base, "crush.json")
	existing, err := common.ReadExistingFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cfgPath, err)
	}
	overlay := map[string]any{}
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

// Import reads crush config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
