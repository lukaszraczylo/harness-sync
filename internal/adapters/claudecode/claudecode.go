// Package claudecode is the claude-code harness adapter.
package claudecode

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/render"
)

const name = "claude-code"

// Adapter implements adapter.Adapter for the claude-code harness.
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

// Detect returns true when ~/.claude exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".claude"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to claude-code.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.home, ".claude")

	fs.Add(adapter.File{
		Dest:          filepath.Join(base, "skills"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "skills"),
	})
	fs.Add(adapter.File{
		Dest:          filepath.Join(base, "agents"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "agents"),
	})

	instructions := b.Instructions.Global
	if override, ok := b.Instructions.PerHarness[name]; ok && override != "" {
		instructions = override
	}
	fs.Add(adapter.File{
		Dest:    filepath.Join(base, "CLAUDE.md"),
		Kind:    adapter.RenderedFile,
		Content: []byte(instructions),
	})

	settings, err := renderSettings(b)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    filepath.Join(base, "settings.json"),
		Kind:    adapter.RenderedFile,
		Content: settings,
	})

	return fs, nil
}

func renderSettings(b *canonical.Bundle) ([]byte, error) {
	return render.JSON(map[string]any{
		"mcpServers": common.BuildMCPMap(&b.MCP),
	})
}

// Import reads claude-code config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
