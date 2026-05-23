// Package claudecode is the claude-code harness adapter.
package claudecode

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
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
// settings.json is MERGED into any existing file so user-managed keys
// settings.json is MERGED into any existing file so user-managed keys
// (hooks, permissions, env, …) are preserved.
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

	settingsPath := filepath.Join(base, "settings.json")
	existing, _ := os.ReadFile(settingsPath)
	overlay := map[string]any{
		"mcpServers": common.BuildMCPMapStyled(&b.MCP, common.MCPClaudeStyle),
	}
	merged, err := common.MergeJSONKeys(existing, overlay)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    settingsPath,
		Kind:    adapter.RenderedFile,
		Content: merged,
	})

	return fs, nil
}

// Import reads claude-code config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
