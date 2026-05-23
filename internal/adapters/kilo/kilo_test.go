package kilo

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

func TestKiloRender(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root: "/canon",
		Profile: canonical.Profile{
			Name:    "home",
			Gateway: canonical.Gateway{URL: "https://gw", Token: "dummy", DefaultModel: "sonnet"},
			Models:  []canonical.Model{{ID: "claude-sonnet-4-6", Alias: "sonnet"}},
		},
		MCP: canonical.MCPRegistry{Servers: []canonical.MCPServer{
			{Name: "filepuff", Command: "/bin/filepuff"},
		}},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	agentDest := filepath.Join(home, ".config", "kilo", "agent")
	assert.Equal(t, adapter.SymlinkDir, seen[agentDest].Kind)
	assert.Equal(t, "/canon/agents", seen[agentDest].SymlinkTarget)

	cfgDest := filepath.Join(home, ".config", "kilo", "kilo.json")
	assert.Equal(t, adapter.RenderedFile, seen[cfgDest].Kind)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))
	// model is "providerID/modelID" string.
	assert.Equal(t, "harness-sync-gateway/sonnet", parsed["model"])
	assert.Equal(t, "harness-sync-gateway/sonnet", parsed["small_model"])
	// provider map present with npm field.
	provider, ok := parsed["provider"].(map[string]any)
	require.True(t, ok, "provider must be a map")
	gw, ok := provider["harness-sync-gateway"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "@ai-sdk/openai-compatible", gw["npm"])
	assert.Contains(t, parsed, "mcp")
	assert.NotContains(t, parsed, "default_model")
	assert.NotContains(t, parsed, "mcpServers")
	// MCP entries use local/remote type (MCPOpencodeStyle).
	mcp := parsed["mcp"].(map[string]any)
	fp := mcp["filepuff"].(map[string]any)
	assert.Equal(t, "local", fp["type"])
	cmd, _ := fp["command"].([]any)
	assert.Equal(t, "/bin/filepuff", cmd[0])
}

func TestKiloRenderMergesExistingKeys(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "kilo")
	require.NoError(t, os.MkdirAll(base, 0o750))

	existing := map[string]any{
		"$schema":     "https://kilo.schema/v1",
		"small_model": "claude-haiku",
		"compaction":  map[string]any{"enabled": true},
	}
	body, _ := json.Marshal(existing)
	require.NoError(t, os.WriteFile(filepath.Join(base, "kilo.json"), body, 0o600))

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

	cfgDest := filepath.Join(base, "kilo.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))
	// User keys preserved.
	assert.Equal(t, "https://kilo.schema/v1", parsed["$schema"])
	assert.Contains(t, parsed, "compaction")
	// Model updated to providerID/modelID; small_model also set.
	assert.Equal(t, "harness-sync-gateway/sonnet", parsed["model"])
	assert.Equal(t, "harness-sync-gateway/sonnet", parsed["small_model"])
}
func TestKiloRenderEmitsModelString(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{URL: "https://gw", Token: "tok", DefaultModel: "claude-sonnet-4-6"},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "kilo", "kilo.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))

	// model must be "harness-sync-gateway/<modelID>".
	assert.Equal(t, "harness-sync-gateway/claude-sonnet-4-6", parsed["model"])
	assert.Equal(t, "harness-sync-gateway/claude-sonnet-4-6", parsed["small_model"])

	// provider map has npm field.
	provider, ok := parsed["provider"].(map[string]any)
	require.True(t, ok)
	gw, ok := provider["harness-sync-gateway"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "@ai-sdk/openai-compatible", gw["npm"])
	opts, ok := gw["options"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://gw", opts["baseURL"])
}

func TestKiloImport(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "kilo")
	require.NoError(t, os.MkdirAll(base, 0o750))
	body, _ := json.Marshal(map[string]any{"providers": []map[string]any{{"id": "x"}}})
	require.NoError(t, os.WriteFile(filepath.Join(base, "kilo.json"), body, 0o600))

	ad := New(WithHome(home))
	res, err := ad.Import(home)
	require.NoError(t, err)
	assert.NotNil(t, res)
}
