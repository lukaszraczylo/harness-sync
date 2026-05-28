// Package goose is the Goose (Block/square) harness adapter.
//
// Goose uses ~/.config/goose/config.yaml with a flat top-level structure.
// MCP-equivalent servers go under "extensions" (not "mcp").
// Model is split into GOOSE_PROVIDER + GOOSE_MODEL.
//
// When a gateway URL is configured we:
//   - Write ~/.config/goose/custom_providers/<providerName>.json
//   - Set GOOSE_PROVIDER to <providerName> in config.yaml
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
	a := &Adapter{home: common.DefaultHome()}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Name returns the harness identifier.
func (a *Adapter) Name() string { return name }

// Capabilities declares what goose harness-sync manages.
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

// Detect returns true when ~/.config/goose/ exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".config", "goose"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to goose.
// config.yaml is MERGED; only GOOSE_PROVIDER, GOOSE_MODEL, and the
// managed extension entries are overlaid.
// When Gateway.URL is set, a custom_providers/<providerName>.json file is
// also added to the FileSet so goose can load the provider on first run.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	cfgPath := filepath.Join(a.home, ".config", "goose", "config.yaml")

	// goose reads skills from ~/.agents/skills/<name>/SKILL.md (open Agent Skills spec)
	fs.Add(adapter.File{
		Dest:          filepath.Join(a.home, ".agents", "skills"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "skills"),
	})

	existing, _ := os.ReadFile(cfgPath)

	provider, model := splitProviderModel(b.Profile.Gateway.DefaultModel)
	if b.Profile.Gateway.URL != "" {
		// Emit the custom provider JSON file.
		body, providerName := common.GooseCustomProviderFile(&b.Profile)
		if body != nil {
			cpDir := filepath.Join(a.home, ".config", "goose", "custom_providers")
			fs.Add(adapter.File{
				Dest:    filepath.Join(cpDir, providerName+".json"),
				Kind:    adapter.RenderedFile,
				Content: body,
			})
			provider = providerName
		}
	}

	overlay := map[string]any{
		"GOOSE_PROVIDER": provider,
		"GOOSE_MODEL":    model,
	}

	if extensions := buildExtensions(b.MCP.Servers); len(extensions) > 0 {
		// Union with existing so user-added extensions survive an apply.
		overlay["extensions"] = common.UnionNestedMapYAML(existing, "extensions", extensions)
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
// Remote (URL) servers become a remote extension instead of a broken stdio
// entry; servers with neither command nor URL are skipped (nothing launchable).
func buildExtensions(servers []canonical.MCPServer) map[string]any {
	out := map[string]any{}
	for _, s := range servers {
		e := map[string]any{"enabled": true}
		switch {
		case s.URL != "":
			// Remote extension. Goose uses type sse|streamable_http with the
			// endpoint under "uri". NEEDS-VERIFICATION against goose's current
			// extension schema; honour Transport when the profile sets it.
			t := s.Transport
			if t == "" {
				t = "sse"
			}
			e["type"] = t
			e["uri"] = s.URL
		case s.Command != "":
			e["type"] = "stdio"
			e["cmd"] = s.Command
			args := s.Args
			if args == nil {
				args = []string{}
			}
			e["args"] = args
		default:
			// No URL and no command: not launchable — skip rather than emit a
			// stdio entry with an empty cmd.
			continue
		}
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
	if p, m, ok := strings.Cut(s, "/"); ok {
		return p, m
	}
	return "anthropic", s
}

// Import reads goose config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
