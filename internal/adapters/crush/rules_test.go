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

func renderCrush(t *testing.T, home string, b *canonical.Bundle) map[string]adapter.File {
	t.Helper()
	fs, err := New(WithHome(home)).Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })
	return seen
}

func TestCrushWritesAGENTSAndRegistersContextPath(t *testing.T) {
	home := t.TempDir()
	b := &canonical.Bundle{
		Root:         "/canon",
		Instructions: canonical.Instructions{Global: "# global"},
		Rules:        []canonical.Rule{{Name: "sec", Body: "# Security\n\nno secrets in code"}},
	}
	seen := renderCrush(t, home, b)

	base := filepath.Join(home, ".config", "crush")
	agentsMD := filepath.Join(base, "AGENTS.md")
	assert.Equal(t, adapter.RenderedFile, seen[agentsMD].Kind)
	body := string(seen[agentsMD].Content)
	assert.Contains(t, body, "# global")
	assert.Contains(t, body, "no secrets in code")

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(seen[filepath.Join(base, "crush.json")].Content, &cfg))
	opts := cfg["options"].(map[string]any)
	paths := opts["context_paths"].([]any)
	assert.Contains(t, paths, agentsMD, "absolute AGENTS.md path must be registered for global load")
}

func TestCrushPreservesExistingOptionsAndIsIdempotent(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "crush")
	require.NoError(t, os.MkdirAll(base, 0o750))
	agentsMD := filepath.Join(base, "AGENTS.md")

	// Existing crush.json with other options keys and a pre-existing context path
	// (including our own — second apply must not duplicate it).
	existing := map[string]any{
		"options": map[string]any{
			"context_paths": []any{"PROJECT.md", agentsMD},
			"lsp":           map[string]any{"go": true},
		},
	}
	body, _ := json.Marshal(existing)
	require.NoError(t, os.WriteFile(filepath.Join(base, "crush.json"), body, 0o600))

	seen := renderCrush(t, home, &canonical.Bundle{Root: "/canon"})

	var cfg map[string]any
	require.NoError(t, json.Unmarshal(seen[filepath.Join(base, "crush.json")].Content, &cfg))
	opts := cfg["options"].(map[string]any)

	// Other option keys survive.
	assert.Contains(t, opts, "lsp")
	// Pre-existing user path survives; ours is present exactly once (idempotent).
	paths := opts["context_paths"].([]any)
	count := 0
	for _, p := range paths {
		if p == agentsMD {
			count++
		}
	}
	assert.Equal(t, 1, count, "our context path must not be duplicated on re-apply")
	assert.Contains(t, paths, "PROJECT.md")
}
