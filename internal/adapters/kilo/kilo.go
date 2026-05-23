// Package kilo is the kilo harness adapter.
package kilo

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/render"
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

// Detect returns true when ~/.config/kilo exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".config", "kilo"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to kilo.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.home, ".config", "kilo")

	fs.Add(adapter.File{
		Dest:          filepath.Join(base, "agent"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "agents"),
	})

	cfg, err := renderConfig(b)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    filepath.Join(base, "kilo.json"),
		Kind:    adapter.RenderedFile,
		Content: cfg,
	})

	return fs, nil
}

func renderConfig(b *canonical.Bundle) ([]byte, error) {
	out := map[string]any{
		"model":     b.Profile.Gateway.DefaultModel,
		"providers": common.BuildProviders(&b.Profile),
	}
	if mcp := common.BuildMCPMap(&b.MCP); len(mcp) > 0 {
		out["mcp"] = mcp
	}
	return render.JSON(out)
}

// Import reads kilo config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
