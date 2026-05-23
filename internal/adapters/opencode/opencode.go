// Package opencode is the opencode harness adapter.
package opencode

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/render"
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
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.home, ".config", "opencode")

	cfg, err := renderConfig(b)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    filepath.Join(base, "opencode.jsonc"),
		Kind:    adapter.RenderedFile,
		Content: cfg,
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

func renderConfig(b *canonical.Bundle) ([]byte, error) {
	out := map[string]any{
		"providers":     common.BuildProviders(&b.Profile),
		"default_model": b.Profile.Gateway.DefaultModel,
	}
	if mcp := common.BuildMCPMap(&b.MCP); len(mcp) > 0 {
		out["mcpServers"] = mcp
	}
	return render.JSON(out)
}

// Import reads opencode config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
