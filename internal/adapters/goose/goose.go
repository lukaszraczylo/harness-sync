// Package goose is the Goose (Block/square) harness adapter.
//
// Goose uses ~/.config/goose/config.yaml with a flat top-level structure.
// MCP-equivalent servers go under "extensions" (not "mcp").
// Model is split into GOOSE_PROVIDER + GOOSE_MODEL.
//
// Custom provider files (~/.config/goose/custom_providers/<name>.yaml) are
// out of v1 scope. If a gateway URL is configured we set GOOSE_PROVIDER to
// signal the intent; the user must hand-write the custom_providers file.
package goose

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "goose"

// Adapter implements adapter.Adapter for the Goose harness.
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

// Detect returns true when ~/.config/goose/ exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".config", "goose"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to goose.
// config.yaml is MERGED; only GOOSE_PROVIDER, GOOSE_MODEL, and the
// managed extension entries are overlaid.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	cfgPath := filepath.Join(a.home, ".config", "goose", "config.yaml")

	existing, _ := os.ReadFile(cfgPath)

	provider, model := splitProviderModel(b.Profile.Gateway.DefaultModel)
	if b.Profile.Gateway.URL != "" {
		// Custom gateway: signal with provider name. User must create
		// ~/.config/goose/custom_providers/harness-sync.yaml manually.
		provider = "harness-sync"
	}

	overlay := map[string]any{
		"GOOSE_PROVIDER": provider,
		"GOOSE_MODEL":    model,
	}

	if len(b.MCP.Servers) > 0 {
		extensions := buildExtensions(b.MCP.Servers)
		overlay["extensions"] = extensions
	}

	merged, err := common.MergeYAMLKeys(existing, overlay)
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

// buildExtensions converts canonical MCP servers to goose extension entries.
func buildExtensions(servers []canonical.MCPServer) map[string]any {
	out := map[string]any{}
	for _, s := range servers {
		e := map[string]any{
			"enabled": true,
			"type":    "stdio",
		}
		if s.Command != "" {
			e["cmd"] = s.Command
		}
		args := s.Args
		if args == nil {
			args = []string{}
		}
		e["args"] = args
		if len(s.Env) > 0 {
			e["env"] = s.Env
		}
		out[s.Name] = e
	}
	return out
}

// splitProviderModel splits "provider/model" into its parts.
// If no slash, provider is "anthropic" and the whole string is the model.
func splitProviderModel(s string) (string, string) {
	if idx := strings.IndexByte(s, '/'); idx >= 0 {
		return s[:idx], s[idx+1:]
	}
	return "anthropic", s
}

// Import reads goose config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
