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
			Gateway: canonical.Gateway{DefaultModel: "anthropic/claude-sonnet-4-6"},
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

	// Agent default_model is an object.
	agent := parsed["agent"].(map[string]any)
	dm := agent["default_model"].(map[string]any)
	assert.Equal(t, "anthropic", dm["provider"])
	assert.Equal(t, "claude-sonnet-4-6", dm["model"])

	// context_servers.
	cs := parsed["context_servers"].(map[string]any)
	fp := cs["filepuff"].(map[string]any)
	assert.Equal(t, true, fp["enabled"])
	assert.Equal(t, "custom", fp["source"])
	assert.Equal(t, "/bin/filepuff", fp["command"])
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
	// Managed keys present.
	assert.Contains(t, parsed, "agent")
	assert.Contains(t, parsed, "context_servers")
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
	// context_servers key present but empty map.
	cs, ok := parsed["context_servers"]
	assert.True(t, ok)
	assert.Empty(t, cs)
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

func TestSplitProviderModel(t *testing.T) {
	t.Run("with slash", func(t *testing.T) {
		p, m := splitProviderModel("anthropic/claude-sonnet-4-6")
		assert.Equal(t, "anthropic", p)
		assert.Equal(t, "claude-sonnet-4-6", m)
	})
	t.Run("no slash", func(t *testing.T) {
		p, m := splitProviderModel("claude-sonnet")
		assert.Equal(t, "anthropic", p)
		assert.Equal(t, "claude-sonnet", m)
	})
}
