package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestOpencodeRendersAgentsSymlinkAndFoldsRules(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root:         "/canon",
		Instructions: canonical.Instructions{Global: "# global"},
		Rules:        []canonical.Rule{{Name: "go", Body: "# Go\n\ngofmt always"}},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	agentsDest := filepath.Join(home, ".config", "opencode", "agents")
	assert.Equal(t, adapter.SymlinkDir, seen[agentsDest].Kind)
	assert.Equal(t, "/canon/agents", seen[agentsDest].SymlinkTarget)

	agentsMD := filepath.Join(home, ".config", "opencode", "AGENTS.md")
	assert.Equal(t, adapter.RenderedFile, seen[agentsMD].Kind)
	body := string(seen[agentsMD].Content)
	assert.Contains(t, body, "# global")
	assert.Contains(t, body, "gofmt always")
}

func TestOpencodeImportsAgents(t *testing.T) {
	home := t.TempDir()
	agentsDir := filepath.Join(home, ".config", "opencode", "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "review.md"),
		[]byte("---\nname: review\ndescription: d\n---\nbody"), 0o600))

	ad := New(WithHome(home))
	res, err := ad.Import(home)
	require.NoError(t, err)
	require.Len(t, res.Agents, 1)
	assert.Equal(t, "review", res.Agents[0].Name)
}
