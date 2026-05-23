// Package crush is the charmbracelet/crush harness adapter.
package crush

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/render"
)

const name = "crush"

// Adapter implements adapter.Adapter for the crush harness.
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

// Detect returns true when ~/.config/crush exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".config", "crush"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to crush.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.home, ".config", "crush")

	fs.Add(adapter.File{
		Dest:          filepath.Join(base, "skills"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "skills"),
	})

	cfg, err := renderConfig(b)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    filepath.Join(base, "crush.json"),
		Kind:    adapter.RenderedFile,
		Content: cfg,
	})

	return fs, nil
}

func renderConfig(b *canonical.Bundle) ([]byte, error) {
	providers := make([]map[string]any, 0, len(b.Profile.Upstreams)+1)

	// Gateway as primary provider.
	if b.Profile.Gateway.URL != "" {
		gw := map[string]any{
			"id":       "harness-sync-gateway",
			"name":     "harness-sync gateway",
			"base_url": b.Profile.Gateway.URL,
			"api_key":  b.Profile.Gateway.Token,
		}
		models := make([]map[string]any, 0, len(b.Profile.Models))
		for _, m := range b.Profile.Models {
			entry := map[string]any{"id": m.ID}
			if m.Alias != "" {
				entry["alias"] = m.Alias
			}
			models = append(models, entry)
		}
		if len(models) > 0 {
			gw["models"] = models
		}
		providers = append(providers, gw)
	}

	for _, up := range b.Profile.Upstreams {
		p := map[string]any{"id": up.Name, "name": up.Name}
		if up.BaseURL != "" {
			p["base_url"] = up.BaseURL
		}
		if up.APIKey != "" {
			p["api_key"] = up.APIKey
		}
		providers = append(providers, p)
	}

	mcp := map[string]any{}
	for _, s := range b.MCP.Servers {
		entry := map[string]any{}
		if s.Command != "" {
			entry["command"] = s.Command
		}
		if len(s.Args) > 0 {
			entry["args"] = s.Args
		}
		if s.URL != "" {
			entry["url"] = s.URL
		}
		if s.Transport != "" {
			entry["transport"] = s.Transport
		}
		if len(s.Env) > 0 {
			entry["env"] = s.Env
		}
		mcp[s.Name] = entry
	}

	out := map[string]any{
		"providers":     providers,
		"default_model": b.Profile.Gateway.DefaultModel,
	}
	if len(mcp) > 0 {
		out["mcpServers"] = mcp
	}
	return render.JSON(out)
}

// Import reads crush config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
