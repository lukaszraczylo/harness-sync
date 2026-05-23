package common_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/fsx"
)

func TestImportMarkdownTreeFSPicksUpSkillsFromMemFS(t *testing.T) {
	mem := fsx.Mem()

	body := []byte("---\nname: cool-skill\ndescription: does stuff\n---\n# Cool Skill\n")
	require.NoError(t, mem.MkdirAll("/skills/cool-skill", 0o755))
	require.NoError(t, afero.WriteFile(mem, "/skills/cool-skill/SKILL.md", body, 0o644))

	docs, err := common.ImportMarkdownTreeFS(mem, "/skills", "SKILL.md")
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "cool-skill", docs[0].Name)
	assert.Equal(t, "does stuff", docs[0].Description)
	assert.Contains(t, docs[0].Body, "# Cool Skill")
}

func TestImportMarkdownTreeFSHandlesMissingDir(t *testing.T) {
	mem := fsx.Mem()

	docs, err := common.ImportMarkdownTreeFS(mem, "/no-such-dir", "SKILL.md")
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestImportMarkdownTreeFSSkipsNonMatching(t *testing.T) {
	mem := fsx.Mem()

	require.NoError(t, mem.MkdirAll("/skills/a", 0o755))
	require.NoError(t, mem.MkdirAll("/skills/b", 0o755))
	require.NoError(t, afero.WriteFile(mem, "/skills/a/SKILL.md", []byte("---\nname: a\n---\n"), 0o644))
	require.NoError(t, afero.WriteFile(mem, "/skills/b/OTHER.md", []byte("# other"), 0o644))
	require.NoError(t, afero.WriteFile(mem, "/skills/b/notes.txt", []byte("text"), 0o644))

	// requiredFilename = "SKILL.md" → only picks up a/SKILL.md
	docs, err := common.ImportMarkdownTreeFS(mem, "/skills", "SKILL.md")
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "a", docs[0].Name)
}

func TestImportMarkdownTreeFSAnyMDWhenNoRequired(t *testing.T) {
	mem := fsx.Mem()

	require.NoError(t, mem.MkdirAll("/agents", 0o755))
	require.NoError(t, afero.WriteFile(mem, "/agents/alpha.md", []byte("# Alpha\n"), 0o644))
	require.NoError(t, afero.WriteFile(mem, "/agents/beta.md", []byte("# Beta\n"), 0o644))
	require.NoError(t, afero.WriteFile(mem, "/agents/skip.txt", []byte("not md"), 0o644))

	docs, err := common.ImportMarkdownTreeFS(mem, "/agents", "")
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}
