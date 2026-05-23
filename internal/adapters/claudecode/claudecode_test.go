package claudecode

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

	settingsDest := filepath.Join(home, ".claude", "settings.json")
	assert.Equal(t, adapter.RenderedFile, seen[settingsDest].Kind)
	settingsBody := string(seen[settingsDest].Content)
	assert.Contains(t, settingsBody, "filepuff")
	assert.Contains(t, settingsBody, "/bin/filepuff")
}
func TestRenderMergesExistingSettings(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".claude")
	require.NoError(t, os.MkdirAll(base, 0o750))

	// Existing settings.json with user-managed keys.
	existing := map[string]any{
		"hooks":       map[string]any{"preToolUse": []string{"/bin/hook"}},
		"permissions": map[string]any{"allow": []string{"Bash"}},
		"env":         map[string]any{"ANTHROPIC_DEFAULT_MODEL": "opus"},
	}
	body, _ := json.Marshal(existing)
	require.NoError(t, os.WriteFile(filepath.Join(base, "settings.json"), body, 0o600))

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

	settingsDest := filepath.Join(base, "settings.json")
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[settingsDest].Content, &parsed))

	// User keys preserved.
	assert.Contains(t, parsed, "hooks")
	assert.Contains(t, parsed, "permissions")
	assert.Contains(t, parsed, "env")
	// MCP updated.
	mcp := parsed["mcpServers"].(map[string]any)
	assert.Contains(t, mcp, "filepuff")
	// MCP entries must NOT have a "type" field (claude-code style).
	fp := mcp["filepuff"].(map[string]any)
	assert.NotContains(t, fp, "type")
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

	settings, _ := json.Marshal(map[string]any{
		"mcpServers": map[string]any{
			"filepuff": map[string]any{
				"command":   "/bin/filepuff",
				"transport": "stdio",
			},
		},
	})
	require.NoError(t, os.WriteFile(
		filepath.Join(base, "settings.json"), settings, 0o600))

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
