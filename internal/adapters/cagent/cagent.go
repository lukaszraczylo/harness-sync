// Package cagent is the cagent harness adapter.
//
// cagent is per-run (no persistent agent state). v1 renders a starter
// ~/.config/cagent/default.yaml with a single default agent pointing at the
// gateway. Import always returns an empty result.
package cagent

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"

	"gopkg.in/yaml.v3"
)

const name = "cagent"

// Adapter implements adapter.Adapter for the cagent harness.
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

// Capabilities declares what cagent harness-sync manages.
func (a *Adapter) Capabilities() adapter.HarnessCapabilities {
	return adapter.HarnessCapabilities{
		ManagesProviders:    true,
		ManagesModels:       true,
		ManagesMCP:          true,
		ManagesSkills:       false,
		ManagesInstructions: true,
		HasBuiltInSub:       false,
	}
}

// Detect returns true when ~/.config/cagent/ exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".config", "cagent"))
	return err == nil
}

// Render produces a starter default.yaml for cagent.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	cfgPath := filepath.Join(a.home, ".config", "cagent", "default.yaml")

	instructions := b.Instructions.Global
	if override, ok := b.Instructions.PerHarness[name]; ok && override != "" {
		instructions = override
	}

	mcps := map[string]any{}
	for _, s := range b.MCP.Servers {
		e := map[string]any{}
		if s.Command != "" {
			e["command"] = s.Command
		}
		args := s.Args
		if args == nil {
			args = []string{}
		}
		e["args"] = args
		mcps[s.Name] = e
	}

	// inline model shorthand: "providerID/modelID"
	modelRef := ""
	if b.Profile.Gateway.DefaultModel != "" {
		modelRef = "harness-sync-gateway/" + b.Profile.Gateway.DefaultModel
	}

	cfg := map[string]any{
		"version": 8,
		"metadata": map[string]any{
			"name": "harness-sync",
		},
		"agents": map[string]any{
			"default": map[string]any{
				"model":       modelRef,
				"instruction": instructions,
			},
		},
		"providers": common.ProvidersAsCagentMap(&b.Profile),
		"models": map[string]any{
			"harness-sync-gateway": map[string]any{
				"provider": "harness-sync-gateway",
				"model":    b.Profile.Gateway.DefaultModel,
			},
		},
	}
	if len(mcps) > 0 {
		cfg["mcps"] = mcps
	}

	body, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	fs.Add(adapter.File{
		Dest:    cfgPath,
		Kind:    adapter.RenderedFile,
		Content: body,
	})

	return fs, nil
}

// Import returns an empty result — cagent has no persistent state to import.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return &adapter.ImportResult{}, nil
}
