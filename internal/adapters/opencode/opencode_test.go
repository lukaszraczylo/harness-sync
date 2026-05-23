package opencode

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

func TestOpencodeRender(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root: "/canon",
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{URL: "https://gw", Token: "dummy", DefaultModel: "sonnet"},
			Models:  []canonical.Model{{ID: "claude-sonnet-4-6", Alias: "sonnet"}},
		},
		Instructions: canonical.Instructions{Global: "# global"},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	assert.Equal(t, adapter.RenderedFile, seen[cfgDest].Kind)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))
	// Correct key is "provider" (singular map), not "providers" (slice).
	assert.Contains(t, parsed, "provider")
	assert.NotContains(t, parsed, "providers")
	// Correct key is "model", not "default_model".
	assert.Contains(t, parsed, "model")
	assert.NotContains(t, parsed, "default_model")
	// No "mcpServers" — opencode uses "mcp".
	assert.NotContains(t, parsed, "mcpServers")

	agentsDest := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	assert.Equal(t, adapter.RenderedFile, seen[agentsDest].Kind)
	assert.Contains(t, string(seen[agentsDest].Content), "# global")
}

func TestOpencodeRenderWithMCP(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
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

	cfgDest := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))
	mcp := parsed["mcp"].(map[string]any)
	fp := mcp["fp"].(map[string]any)
	// MCPOpencodeStyle: type local, command as array, enabled true.
	assert.Equal(t, "local", fp["type"])
	assert.Equal(t, true, fp["enabled"])
	cmd, _ := fp["command"].([]any)
	assert.Equal(t, "/bin/fp", cmd[0])
}

func TestOpencodeRenderMergesExistingKeys(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "opencode")
	require.NoError(t, os.MkdirAll(base, 0o750))

	existing := map[string]any{
		"$schema": "https://opencode.schema/v1",
		"agent":   map[string]any{"maxTokens": 8192},
	}
	body, _ := json.Marshal(existing)
	require.NoError(t, os.WriteFile(filepath.Join(base, "opencode.jsonc"), body, 0o600))

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

	cfgDest := filepath.Join(base, "opencode.jsonc")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[cfgDest].Content, &parsed))
	// User keys preserved.
	assert.Equal(t, "https://opencode.schema/v1", parsed["$schema"])
	assert.Contains(t, parsed, "agent")
	// Managed keys overlaid.
	assert.Contains(t, parsed, "model")
	assert.Contains(t, parsed, "provider")
}
func TestOpencodeImportStripsComments(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "opencode")
	require.NoError(t, os.MkdirAll(base, 0o750))
	jsonc := `// header comment
{
  "providers": [{"id": "anthropic"}], // trailing
  /* block
     comment */
  "default_model": "sonnet"
}`
	require.NoError(t, os.WriteFile(filepath.Join(base, "opencode.jsonc"), []byte(jsonc), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(base, "AGENTS.md"), []byte("# global"), 0o600))

	ad := New(WithHome(home))
	res, err := ad.Import(home)
	require.NoError(t, err)
	assert.Equal(t, "# global", res.Instructions)
}
