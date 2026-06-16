package goose

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestGooseFoldsRulesIntoGoosehints(t *testing.T) {
	home := t.TempDir()
	b := &canonical.Bundle{
		Instructions: canonical.Instructions{Global: "# global"},
		Rules:        []canonical.Rule{{Name: "k8s", Body: "# Kube\n\nlabel everything"}},
	}
	fs, err := New(WithHome(home)).Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	hints := filepath.Join(home, ".config", "goose", ".goosehints")
	require.Contains(t, seen, hints)
	assert.Equal(t, adapter.RenderedFile, seen[hints].Kind)
	body := string(seen[hints].Content)
	assert.Contains(t, body, "# global")
	assert.Contains(t, body, "label everything")
}
