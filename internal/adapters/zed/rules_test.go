package zed

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestZedFoldsRulesIntoAGENTS(t *testing.T) {
	home := t.TempDir()
	b := &canonical.Bundle{
		Instructions: canonical.Instructions{Global: "# global"},
		Rules:        []canonical.Rule{{Name: "ts", Body: "# TS\n\nstrict mode"}},
	}
	fs, err := New(WithHome(home)).Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	agentsMD := filepath.Join(home, ".config", "zed", "AGENTS.md")
	require.Contains(t, seen, agentsMD)
	assert.Equal(t, adapter.RenderedFile, seen[agentsMD].Kind)
	body := string(seen[agentsMD].Content)
	assert.Contains(t, body, "# global")
	assert.Contains(t, body, "strict mode")
}
