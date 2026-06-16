package kilo

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestKiloFoldsRulesIntoAGENTS(t *testing.T) {
	home := t.TempDir()
	b := &canonical.Bundle{
		Root:         "/canon",
		Instructions: canonical.Instructions{Global: "# global"},
		Rules:        []canonical.Rule{{Name: "py", Body: "# Py\n\ntype hints"}},
	}
	fs, err := New(WithHome(home)).Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	agentsMD := filepath.Join(home, ".config", "kilo", "AGENTS.md")
	require.Contains(t, seen, agentsMD)
	assert.Equal(t, adapter.RenderedFile, seen[agentsMD].Kind)
	body := string(seen[agentsMD].Content)
	assert.Contains(t, body, "# global")
	assert.Contains(t, body, "type hints")
}
