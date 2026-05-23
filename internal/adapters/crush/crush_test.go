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
	// Must use "default_model", not "model".
	assert.Contains(t, parsed, "default_model")
	// Must not write "mcpServers" — crush uses "mcp".
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
