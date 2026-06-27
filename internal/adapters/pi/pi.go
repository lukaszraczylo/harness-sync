// Package pi is the Pi coding agent harness adapter.
package pi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "pi"

// Adapter implements adapter.Adapter for the Pi coding agent harness.
type Adapter struct {
	*adapter.Base
}

// WithHome overrides the home directory used to resolve target paths.
func WithHome(h string) adapter.BaseOption { return func(b *adapter.Base) { b.Home = h } }

// New returns a new Adapter with the given options applied.
func New(opts ...adapter.BaseOption) *Adapter {
	return &Adapter{Base: adapter.NewBase(opts...)}
}

// Name returns the harness identifier.
func (a *Adapter) Name() string { return name }

// Capabilities declares what pi harness-sync manages.
func (a *Adapter) Capabilities() adapter.HarnessCapabilities {
	return adapter.HarnessCapabilities{
		ManagesProviders:    true,
		ManagesModels:       true,
		ManagesMCP:          false,
		ManagesSkills:       true,
		ManagesAgents:       false,
		ManagesRules:        true,
		ManagesInstructions: true,
		HasBuiltInSub:       false,
	}
}

// Detect returns true when ~/.pi/agent exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.Home, ".pi", "agent"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to Pi.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.Home, ".pi", "agent")

	fs.Add(adapter.File{
		Dest:          filepath.Join(base, "skills"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "skills"),
	})

	if instr := b.InstructionTextWithRules(name); instr != "" {
		fs.Add(adapter.File{
			Dest:    filepath.Join(base, "AGENTS.md"),
			Kind:    adapter.RenderedFile,
			Content: []byte(instr),
		})
	}

	cfgPath := filepath.Join(base, "settings.json")
	existing, err := common.ReadExistingFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cfgPath, err)
	}

	overlay := map[string]any{}
	if b.Profile.Gateway.DefaultModel != "" {
		overlay["defaultProvider"] = common.GatewayProviderKey(b.Profile.Gateway.URL)
		overlay["defaultModel"] = b.Profile.Gateway.DefaultModel
	}
	if len(overlay) > 0 {
		merged, err := common.MergeJSONKeys(existing, overlay)
		if err != nil {
			return nil, err
		}
		fs.Add(adapter.File{
			Dest:    cfgPath,
			Kind:    adapter.RenderedFile,
			Content: merged,
		})
	}

	if modelsContent, err := piModelsJSON(b); err != nil {
		return nil, err
	} else if len(modelsContent) > 0 {
		fs.Add(adapter.File{
			Dest:    filepath.Join(base, "models.json"),
			Kind:    adapter.RenderedFile,
			Content: modelsContent,
		})
	}

	return fs, nil
}

func piModelsJSON(b *canonical.Bundle) ([]byte, error) {
	if b.Profile.Gateway.URL == "" {
		return nil, nil
	}

	modelEntries := make([]map[string]any, 0, len(b.Profile.Models))
	for _, m := range b.Profile.Models {
		entry := map[string]any{"id": m.ID}
		if m.Alias != "" {
			entry["name"] = m.Alias
		}
		modelEntries = append(modelEntries, entry)
	}
	if len(modelEntries) == 0 && b.Profile.Gateway.DefaultModel != "" {
		modelEntries = append(modelEntries, map[string]any{"id": b.Profile.Gateway.DefaultModel})
	}

	provider := map[string]any{
		"baseUrl": b.Profile.Gateway.URL,
		"api":     "openai-completions",
		"apiKey":  b.Profile.Gateway.Token,
	}
	if len(modelEntries) > 0 {
		provider["models"] = modelEntries
	}

	body, err := json.MarshalIndent(map[string]any{
		"providers": map[string]any{
			common.GatewayProviderKey(b.Profile.Gateway.URL): provider,
		},
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

// Import reads Pi config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
