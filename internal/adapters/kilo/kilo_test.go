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
	assert.Equal(t, "sonnet", parsed["model"])
	assert.Contains(t, parsed, "mcp")
	assert.NotContains(t, parsed, "default_model")
	assert.NotContains(t, parsed, "mcpServers")
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
