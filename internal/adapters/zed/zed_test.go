package zed

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

func bundle() *canonical.Bundle {
	return &canonical.Bundle{
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{
				URL:          "https://llmgw.example.com/v1",
				DefaultModel: "claude-sonnet-4-6",
			},
			Models: []canonical.Model{
				{ID: "claude-sonnet-4-6", Alias: "sonnet"},
			},
		},
		MCP: canonical.MCPRegistry{Servers: []canonical.MCPServer{
			{Name: "filepuff", Command: "/bin/filepuff", Args: []string{"--serve"}},
		}},
	}
}

func TestZedRenderProducesExpectedTargets(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	fs, err := ad.Render(bundle())
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "zed", "settings.json")
	require.Contains(t, seen, cfgDest)
	assert.Equal(t, adapter.RenderedFile, seen[cfgDest].Kind)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))

	// agent.default_model routes through harness-sync-gateway (openai_compatible provider).
	agent := parsed["agent"].(map[string]any)
	dm := agent["default_model"].(map[string]any)
	assert.Equal(t, "harness-sync-gateway", dm["provider"])
	assert.Equal(t, "claude-sonnet-4-6", dm["model"])

	// language_models.openai_compatible has our named gateway entry.
	lm := parsed["language_models"].(map[string]any)
	oc := lm["openai_compatible"].(map[string]any)
	gw := oc["harness-sync-gateway"].(map[string]any)
	assert.Equal(t, "https://llmgw.example.com/v1", gw["api_url"])

	// context_servers must be absent — Zed serde untagged-enum bug aborts
	// settings loading when this key is present; we delete it via nil overlay.
	assert.NotContains(t, parsed, "context_servers")
}

func TestZedRenderMergesExistingKeys(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "zed")
	require.NoError(t, os.MkdirAll(base, 0o750))

	// Existing settings with unrelated user config.
	existing := map[string]any{
		"theme":     "One Dark",
		"font_size": 16,
		"vim_mode":  true,
	}
	body, _ := json.Marshal(existing)
	require.NoError(t, os.WriteFile(filepath.Join(base, "settings.json"), body, 0o600))

	ad := New(WithHome(home))
	fs, err := ad.Render(bundle())
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(base, "settings.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))

	// Unrelated keys preserved.
	assert.Equal(t, "One Dark", parsed["theme"])
	assert.Equal(t, float64(16), parsed["font_size"])
	assert.Equal(t, true, parsed["vim_mode"])
	// Managed keys present; context_servers absent (serde bug — we delete it).
	assert.Contains(t, parsed, "agent")
	assert.NotContains(t, parsed, "context_servers")
}

func TestZedRenderNoMCP(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{DefaultModel: "anthropic/sonnet"},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "zed", "settings.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))
	// context_servers must be absent (nil overlay deletes the key).
	assert.NotContains(t, parsed, "context_servers")
}

func TestZedRenderEmitsLanguageModelsOpenAICompatible(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{URL: "https://gw", Token: "tok", DefaultModel: "claude-sonnet-4-6"},
			Models:  []canonical.Model{{ID: "claude-sonnet-4-6", Alias: "sonnet"}},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "zed", "settings.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))

	lm, ok := parsed["language_models"].(map[string]any)
	require.True(t, ok, "language_models must be present")
	oc, ok := lm["openai_compatible"].(map[string]any)
	require.True(t, ok, "language_models.openai_compatible must be present")
	gw, ok := oc["harness-sync-gateway"].(map[string]any)
	require.True(t, ok, "openai_compatible.harness-sync-gateway must be present")
	assert.Equal(t, "https://gw", gw["api_url"])
	avail, ok := gw["available_models"].([]any)
	require.True(t, ok)
	require.Len(t, avail, 1)
	m0 := avail[0].(map[string]any)
	assert.Equal(t, "claude-sonnet-4-6", m0["name"])
	assert.Equal(t, "sonnet", m0["display_name"])
	caps, ok := m0["capabilities"].(map[string]any)
	require.True(t, ok, "model must have capabilities")
	assert.Equal(t, true, caps["tools"])
	assert.Equal(t, false, caps["prompt_cache_key"])
}

func TestZedImport(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "zed")
	require.NoError(t, os.MkdirAll(base, 0o750))

	settings := map[string]any{
		"context_servers": map[string]any{
			"filepuff": map[string]any{
				"enabled": true,
				"source":  "custom",
				"command": "/bin/filepuff",
				"args":    []string{"--serve"},
			},
		},
	}
	body, _ := json.Marshal(settings)
	require.NoError(t, os.WriteFile(filepath.Join(base, "settings.json"), body, 0o600))

	ad := New(WithHome(home))
	res, err := ad.Import(home)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.MCP, 1)
	assert.Equal(t, "filepuff", res.MCP[0].Name)
	assert.Equal(t, "/bin/filepuff", res.MCP[0].Command)
}

func TestZedImportMissing(t *testing.T) {
	ad := New(WithHome(t.TempDir()))
	res, err := ad.Import(t.TempDir())
	require.NoError(t, err)
	assert.NotNil(t, res)
}

func TestZedDetect(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	assert.False(t, ad.Detect())
}
