// Package opencode is the opencode harness adapter.
package opencode

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "opencode"

// Adapter implements adapter.Adapter for the opencode harness.
type Adapter struct{ home string }

// Option configures an Adapter.
type Option func(*Adapter)

// WithHome overrides the home directory (defaults to os.UserHomeDir).
func WithHome(h string) Option { return func(a *Adapter) { a.home = h } }

// New returns a new Adapter with the given options applied.
func New(opts ...Option) *Adapter {
	a := &Adapter{home: defaultHome()}
	for _, o := range opts {
		o(a)
	}
	return a
}

func defaultHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// Name returns the harness identifier.
func (a *Adapter) Name() string { return name }

// Detect returns true when ~/.config/opencode exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".config", "opencode"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to opencode.
// opencode.jsonc is MERGED to preserve user-managed keys ($schema, agent,
// instructions, …). Correct keys: "provider" (singular, map), "model" (string),
// "mcp" (map with type: local|remote entries).
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.home, ".config", "opencode")

	cfgPath := filepath.Join(base, "opencode.jsonc")
	existing, _ := os.ReadFile(cfgPath)
	overlay := map[string]any{
		"provider": common.ProvidersAsMap(&b.Profile),
		"model":    b.Profile.Gateway.DefaultModel,
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
