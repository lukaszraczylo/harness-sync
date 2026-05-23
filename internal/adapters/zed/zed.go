// Package zed is the Zed editor harness adapter.
//
// Zed uses ~/.config/zed/settings.json with many user-managed keys.
// We MERGE only "agent" and "context_servers"; all other keys are preserved.
// MCP servers become context_servers entries with source: "custom".
package zed

import (
	"os"
	"path/filepath"
	"strings"

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

// Detect returns true when ~/.config/zed/ exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".config", "zed"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to Zed.
// settings.json is MERGED to preserve all user-managed Zed configuration.
// Only "agent" (default_model) and "context_servers" (MCP) are overlaid.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	cfgPath := filepath.Join(a.home, ".config", "zed", "settings.json")
	existing, _ := os.ReadFile(cfgPath)

	overlay := map[string]any{
		"context_servers": common.BuildMCPMapStyled(&b.MCP, common.MCPZedStyle),
	}

	if dm := b.Profile.Gateway.DefaultModel; dm != "" {
		provider, model := splitProviderModel(dm)
		overlay["agent"] = map[string]any{
			"default_model": map[string]any{
				"provider": provider,
				"model":    model,
			},
		}
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

// Import reads context_servers from Zed settings.json back to MCP servers.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}

// splitProviderModel splits "provider/model" at the first slash.
// If no slash, provider is "anthropic".
func splitProviderModel(s string) (string, string) {
	if idx := strings.IndexByte(s, '/'); idx >= 0 {
		return s[:idx], s[idx+1:]
	}
	return "anthropic", s
}
