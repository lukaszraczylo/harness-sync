package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestRenderPreservesExistingCLAUDEmd(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".claude")
	require.NoError(t, os.MkdirAll(base, 0o750))

	// A user's hand-written CLAUDE.md must not be clobbered.
	userContent := "# My personal Claude instructions\nNever delete my notes.\n"
	require.NoError(t, os.WriteFile(filepath.Join(base, "CLAUDE.md"), []byte(userContent), 0o600))

	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root:         "/canon",
		Instructions: canonical.Instructions{Global: "canonical guidance"},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	got := string(seen[filepath.Join(base, "CLAUDE.md")].Content)
	assert.Contains(t, got, "Never delete my notes.", "user content must be preserved")
	assert.Contains(t, got, "canonical guidance", "managed block must be inserted")
	assert.Contains(t, got, common.ManagedBlockBegin)
}

func TestRenderSkipsEmptyCLAUDEmd(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{Root: "/canon"} // no instructions
	fs, err := ad.Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })
	_, has := seen[filepath.Join(home, ".claude", "CLAUDE.md")]
	assert.False(t, has, "no CLAUDE.md emitted when there are no instructions")
}

func TestRenderProducesExpectedTargets(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root:   "/canon",
		Config: canonical.Config{ActiveProfile: "home"},
		MCP: canonical.MCPRegistry{Servers: []canonical.MCPServer{
			{Name: "filepuff", Command: "/bin/filepuff", Transport: "stdio"},
		}},
		Instructions: canonical.Instructions{Global: "# global"},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	skillsDest := filepath.Join(home, ".claude", "skills")
	assert.Equal(t, adapter.SymlinkDir, seen[skillsDest].Kind)
	assert.Equal(t, "/canon/skills", seen[skillsDest].SymlinkTarget)

	agentsDest := filepath.Join(home, ".claude", "agents")
	assert.Equal(t, adapter.SymlinkDir, seen[agentsDest].Kind)
	assert.Equal(t, "/canon/agents", seen[agentsDest].SymlinkTarget)

	claudemdDest := filepath.Join(home, ".claude", "CLAUDE.md")
	assert.Equal(t, adapter.RenderedFile, seen[claudemdDest].Kind)
	assert.Contains(t, string(seen[claudemdDest].Content), "# global")

	mcpDest := filepath.Join(home, ".claude", "mcp_servers.json")
	assert.Equal(t, adapter.RenderedFile, seen[mcpDest].Kind)
	body := string(seen[mcpDest].Content)
	assert.Contains(t, body, "filepuff")
	assert.Contains(t, body, "/bin/filepuff")
	assert.Contains(t, body, "\"type\": \"stdio\"")
}
func TestRenderMergesLiveClaudeJSON(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".claude")
	require.NoError(t, os.MkdirAll(base, 0o750))

	// Existing ~/.claude.json with user-managed top-level keys.
	existing := map[string]any{
		"projects":         map[string]any{"/path": map[string]any{"x": 1}},
		"shellIntegration": true,
		"mcpServers": map[string]any{
			"old": map[string]any{"command": "/bin/old", "type": "stdio"},
		},
	}
	body, _ := json.Marshal(existing)
	require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), body, 0o600))

	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root: "/canon",
		MCP: canonical.MCPRegistry{Servers: []canonical.MCPServer{
			{Name: "filepuff", Command: "/bin/filepuff"},
		}},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	livePath := filepath.Join(home, ".claude.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[livePath].Content, &parsed))

	// User-managed top-level keys preserved.
	assert.Contains(t, parsed, "projects")
	assert.Contains(t, parsed, "shellIntegration")
	// mcpServers is unioned: canonical server added, user-added server preserved
	// (a user's `claude mcp add` entry must survive an apply).
	mcp := parsed["mcpServers"].(map[string]any)
	assert.Contains(t, mcp, "filepuff")
	assert.Contains(t, mcp, "old", "user-added MCP server must be preserved, not dropped")
	// Each entry uses Claude's "type" key.
	fp := mcp["filepuff"].(map[string]any)
	assert.Equal(t, "stdio", fp["type"])
}

func TestClaudeCodeRenderNeverWritesSettings(t *testing.T) {
	// Claude Code uses its own Anthropic Max subscription.
	// harness-sync must NEVER write settings.json even when a gateway is configured.
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root: "/canon",
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{
				URL:          "https://gw",
				Token:        "tok",
				DefaultModel: "claude-sonnet-4-6",
			},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	settingsDest := filepath.Join(home, ".claude", "settings.json")
	assert.NotContains(t, seen, settingsDest, "settings.json must never be written — claude-code has its own subscription")
}

func TestClaudeCodeRenderNoGatewayNoSettings(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root:    "/canon",
		Profile: canonical.Profile{},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	settingsDest := filepath.Join(home, ".claude", "settings.json")
	assert.NotContains(t, seen, settingsDest, "settings.json must not be written when no gateway URL")
}

func TestImport(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(base, "skills", "hello"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(base, "agents"), 0o750))

	require.NoError(t, os.WriteFile(
		filepath.Join(base, "skills", "hello", "SKILL.md"),
		[]byte("---\nname: hello\ndescription: x\n---\nbody"), 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(base, "agents", "agentA.md"),
		[]byte("---\nname: agentA\ndescription: y\n---\nagent body"), 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(base, "CLAUDE.md"),
		[]byte("# global"), 0o600))

	// Live ~/.claude.json with mcpServers (the primary location).
	liveBody, _ := json.Marshal(map[string]any{
		"mcpServers": map[string]any{
			"filepuff": map[string]any{
				"command": "/bin/filepuff",
				"type":    "stdio",
			},
		},
	})
	require.NoError(t, os.WriteFile(
		filepath.Join(home, ".claude.json"), liveBody, 0o600))

	ad := New(WithHome(home))
	res, err := ad.Import(home)
	require.NoError(t, err)

	require.Len(t, res.Skills, 1)
	assert.Equal(t, "hello", res.Skills[0].Name)
	require.Len(t, res.Agents, 1)
	assert.Equal(t, "agentA", res.Agents[0].Name)
	require.Len(t, res.MCP, 1)
	assert.Equal(t, "filepuff", res.MCP[0].Name)
	assert.Equal(t, "/bin/filepuff", res.MCP[0].Command)
	assert.Equal(t, "# global", res.Instructions)
}
