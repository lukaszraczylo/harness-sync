package crush

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestCrushRenderProducesExpectedTargets(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root: "/canon",
		Profile: canonical.Profile{
			Name:    "home",
			Gateway: canonical.Gateway{URL: "https://gw", Token: "dummy", DefaultModel: "sonnet"},
			Models:  []canonical.Model{{ID: "claude-sonnet-4-6", Alias: "sonnet"}},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	skillsDest := filepath.Join(home, ".config", "crush", "skills")
	assert.Equal(t, adapter.SymlinkDir, seen[skillsDest].Kind)
	assert.Equal(t, "/canon/skills", seen[skillsDest].SymlinkTarget)

	cfgDest := filepath.Join(home, ".config", "crush", "crush.json")
	assert.Equal(t, adapter.RenderedFile, seen[cfgDest].Kind)

	// Parse rendered JSON to verify structure.
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))
	assert.NotNil(t, parsed)
	// providers must be a map (not array), keyed by providerID.
	providers, ok := parsed["providers"].(map[string]any)
	require.True(t, ok, "providers must be a map")
	gw, ok := providers["hs-gw"].(map[string]any)
	require.True(t, ok, "hs-gw provider missing")
	assert.Equal(t, "openai-compat", gw["type"])
	// models must be role-map with large/small/title.
	models, ok := parsed["models"].(map[string]any)
	require.True(t, ok, "models must be a map")
	assert.Contains(t, models, "large")
	assert.Contains(t, models, "small")
	assert.Contains(t, models, "title")
	// Must write "default_model" (prevents Bedrock fallback) but NOT "mcpServers".
	assert.Contains(t, parsed, "default_model")
	assert.NotContains(t, parsed, "mcpServers")
}

func TestCrushRenderMCPHasType(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root: "/canon",
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{DefaultModel: "sonnet"},
		},
		MCP: canonical.MCPRegistry{Servers: []canonical.MCPServer{
			{Name: "fp", Command: "/bin/fp"},
		}},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "crush", "crush.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))
	mcp := parsed["mcp"].(map[string]any)
	fp := mcp["fp"].(map[string]any)
	// Each MCP entry must have a "type" field.
	assert.Equal(t, "stdio", fp["type"])
}

func TestCrushRenderMergesExistingKeys(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "crush")
	require.NoError(t, os.MkdirAll(base, 0o750))

	existing := map[string]any{
		"$schema":     "https://crush.schema/v1",
		"lsp":         []string{"gopls"},
		"permissions": map[string]any{"allow": []string{"Bash"}},
	}
	body, _ := json.Marshal(existing)
	require.NoError(t, os.WriteFile(filepath.Join(base, "crush.json"), body, 0o600))

	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{DefaultModel: "sonnet"},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(base, "crush.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))
	// Unrelated user keys preserved.
	assert.Equal(t, "https://crush.schema/v1", parsed["$schema"])
	assert.Contains(t, parsed, "lsp")
	assert.Contains(t, parsed, "permissions")
}
func TestCrushRenderProducesMapProvidersAndRoleModels(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{URL: "https://gw", Token: "tok", DefaultModel: "claude-sonnet-4-6"},
			Models:  []canonical.Model{{ID: "claude-sonnet-4-6"}},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "crush", "crush.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))

	// providers is a map, not an array.
	providers, ok := parsed["providers"].(map[string]any)
	require.True(t, ok, "providers must be a map")
	gw, ok := providers["hs-gw"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "openai-compat", gw["type"])
	assert.Equal(t, "https://gw", gw["base_url"])
	provModels, ok := gw["models"].([]any)
	require.True(t, ok, "models must be array inside provider")
	require.Len(t, provModels, 1)
	m0 := provModels[0].(map[string]any)
	assert.Equal(t, "claude-sonnet-4-6", m0["id"])
	// All 10 required fields present.
	for _, field := range []string{"id", "name", "cost_per_1m_in", "cost_per_1m_out",
		"cost_per_1m_in_cached", "cost_per_1m_out_cached", "context_window",
		"default_max_tokens", "can_reason", "supports_attachments"} {
		assert.Contains(t, m0, field, "missing field %s", field)
	}

	// top-level models is role-map.
	roleModels, ok := parsed["models"].(map[string]any)
	require.True(t, ok)
	for _, role := range []string{"large", "small", "title"} {
		sel, ok := roleModels[role].(map[string]any)
		require.True(t, ok, "role %s missing", role)
		assert.Equal(t, "claude-sonnet-4-6", sel["model"])
		assert.Equal(t, "hs-gw", sel["provider"])
	}
}

func TestCrushImportRoundtrip(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "crush")
	require.NoError(t, os.MkdirAll(base, 0o750))

	// Minimal crush.json
	cfg := map[string]any{
		"providers": []map[string]any{
			{"id": "anthropic", "base_url": "https://api.anthropic.com"},
		},
	}
	body, _ := json.Marshal(cfg)
	require.NoError(t, os.WriteFile(filepath.Join(base, "crush.json"), body, 0o600))

	ad := New(WithHome(home))
	res, err := ad.Import(home)
	require.NoError(t, err)
	assert.NotNil(t, res)
	// Import should return without error even if it can't extract anything.
}
