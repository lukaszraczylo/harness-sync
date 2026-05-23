// Package kilo is the kilo harness adapter.
package kilo

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "kilo"

// Adapter implements adapter.Adapter for the kilo harness.
type Adapter struct {
	home string
}

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

// Capabilities declares what kilo harness-sync manages.
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

// Detect returns true when ~/.config/kilo exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".config", "kilo"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to kilo.
// kilo.json is MERGED to preserve user-managed keys ($schema, small_model,
// instructions, permission, compaction, watcher, formatter, skills).
// Key "model" is a plain string; MCP uses "mcp" with local/remote type entries.
// No "providers" key — kilo has no such concept.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.home, ".config", "kilo")

	fs.Add(adapter.File{
		Dest:          filepath.Join(base, "agent"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "agents"),
	})
	// kilo reads skills from ~/.kilo/skills/<name>/SKILL.md
	fs.Add(adapter.File{
		Dest:          filepath.Join(a.home, ".kilo", "skills"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "skills"),
	})

	cfgPath := filepath.Join(base, "kilo.json")
	existing, _ := os.ReadFile(cfgPath)
	modelStr := common.KiloModelString(&b.Profile)
	overlay := map[string]any{}
	if providers := common.ProvidersAsMap(&b.Profile); len(providers) > 0 {
		overlay["provider"] = providers
	}
	if modelStr != "" {
		overlay["model"] = modelStr
		overlay["small_model"] = modelStr
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

	return fs, nil
}

// Import reads kilo config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
