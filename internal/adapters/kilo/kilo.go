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
	a := &Adapter{home: common.DefaultHome()}
	for _, o := range opts {
		o(a)
	}
	return a
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
// Providers are written under the singular "provider" map (mirroring opencode),
// preserving + deduplicating any the user already configured.
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
	rawProviders := common.ProvidersAsMap(&b.Profile)
	mcpMap := common.BuildMCPMapStyled(&b.MCP, common.MCPOpencodeStyle)

	// Overlay for kilo.json: preserve the user's existing providers (+dedup)
	// and union MCP servers so neither is wholesale-replaced on apply.
	kiloOverlay := map[string]any{}
	if len(rawProviders) > 0 {
		kiloOverlay["provider"] = common.MergeProviderMap(existing, rawProviders, b.Profile.Gateway.URL)
	}
	if modelStr != "" {
		kiloOverlay["model"] = modelStr
		kiloOverlay["small_model"] = modelStr
	}
	if len(mcpMap) > 0 {
		kiloOverlay["mcp"] = common.UnionNestedMap(existing, "mcp", mcpMap)
	}
	merged, err := common.MergeJSONKeys(existing, kiloOverlay)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    cfgPath,
		Kind:    adapter.RenderedFile,
		Content: merged,
	})

	// Also merge into opencode.jsonc when present — kilo reads it as its
	// primary config. Delete the enabled_providers filter so our provider is
	// visible alongside user-defined providers. mergeOpenCodeJSONC merges the
	// raw provider/MCP maps against opencode.jsonc's own existing content.
	ocOverlay := map[string]any{}
	if len(rawProviders) > 0 {
		ocOverlay["provider"] = rawProviders
	}
	if modelStr != "" {
		ocOverlay["model"] = modelStr
		ocOverlay["small_model"] = modelStr
	}
	if len(mcpMap) > 0 {
		ocOverlay["mcp"] = mcpMap
	}
	ocPath := filepath.Join(base, "opencode.jsonc")
	if ocContent, mergeErr := mergeOpenCodeJSONC(ocPath, ocOverlay, b.Profile.Gateway.URL); mergeErr != nil {
		return nil, mergeErr
	} else if ocContent != nil {
		fs.Add(adapter.File{
			Dest:    ocPath,
			Kind:    adapter.RenderedFile,
			Content: ocContent,
			NoMerge: true,
		})
	}

	return fs, nil
}

// mergeOpenCodeJSONC deep-merges our overlay into the existing opencode.jsonc
// file, deleting the enabled_providers filter and deduplicating gateway providers.
// Returns nil content (no error) when the file does not exist.
func mergeOpenCodeJSONC(path string, overlay map[string]any, gatewayURL string) ([]byte, error) {
	ocRaw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil //nolint:nilerr // file absent is not an error
	}
	clean := common.StripJSONComments(string(ocRaw))

	ocOverlay := map[string]any{"enabled_providers": nil}

	if newProv, ok := overlay["provider"].(map[string]any); ok && len(newProv) > 0 {
		ocOverlay["provider"] = common.MergeProviderMap([]byte(clean), newProv, gatewayURL)
	}
	for _, k := range []string{"model", "small_model"} {
		if v, ok := overlay[k]; ok {
			ocOverlay[k] = v
		}
	}
	if mcp, ok := overlay["mcp"].(map[string]any); ok && len(mcp) > 0 {
		ocOverlay["mcp"] = common.UnionNestedMap([]byte(clean), "mcp", mcp)
	}

	return common.MergeJSONKeys([]byte(clean), ocOverlay)
}

// Import reads kilo config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
