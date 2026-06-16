package claudecode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestClaudeCodeRendersRulesSymlink(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{Root: "/canon"}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	rulesDest := filepath.Join(home, ".claude", "rules")
	assert.Equal(t, adapter.SymlinkDir, seen[rulesDest].Kind)
	assert.Equal(t, "/canon/rules", seen[rulesDest].SymlinkTarget)

	// Rules are delivered natively as a directory — never folded into CLAUDE.md.
	claudemd := string(seen[filepath.Join(home, ".claude", "CLAUDE.md")].Content)
	assert.NotContains(t, claudemd, "/canon/rules")
}

func TestClaudeCodeImportsRules(t *testing.T) {
	home := t.TempDir()
	rulesDir := filepath.Join(home, ".claude", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "go.md"),
		[]byte("# Go Rules\n\ngofmt always"), 0o600))

	ad := New(WithHome(home))
	res, err := ad.Import(home)
	require.NoError(t, err)
	require.Len(t, res.Rules, 1)
	assert.Equal(t, "go", res.Rules[0].Name)
	assert.Contains(t, res.Rules[0].Body, "gofmt always")
}
