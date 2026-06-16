package cagent

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"

	"gopkg.in/yaml.v3"
)

func TestCagentFoldsRulesIntoInstruction(t *testing.T) {
	home := t.TempDir()
	b := &canonical.Bundle{
		Instructions: canonical.Instructions{Global: "# global"},
		Rules:        []canonical.Rule{{Name: "rust", Body: "# Rust\n\nno unwrap"}},
	}
	fs, err := New(WithHome(home)).Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfg := seen[filepath.Join(home, ".config", "cagent", "default.yaml")]
	require.Equal(t, adapter.RenderedFile, cfg.Kind)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(cfg.Content, &parsed))
	agents := parsed["agents"].(map[string]any)
	def := agents["default"].(map[string]any)
	instr := def["instruction"].(string)
	assert.Contains(t, instr, "# global")
	assert.Contains(t, instr, "no unwrap")
}
