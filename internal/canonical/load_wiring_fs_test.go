package canonical_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/fsx"
)

// TestLoadFSStrictRejectsUnknownKey: a misspelled key must fail loudly rather
// than silently leaving the field at its zero value (#13).
func TestLoadFSStrictRejectsUnknownKey(t *testing.T) {
	mem := fsx.Mem()
	writeFile(t, mem, "/root/config.yaml", []byte("active_profile: test\n"))
	// An unknown gateway key (here a mistyped field) must be rejected, not
	// silently dropped to leave default_model empty.
	writeFile(t, mem, "/root/profiles/test.yaml",
		[]byte("name: t\ngateway:\n  url: u\n  token: tok\n  no_such_key: m\n"))

	_, err := canonical.LoadFS(mem, "/root")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field names")
}

// TestLoadFSHonoursPaths: Config.Paths relocates the skills source dir (#14).
func TestLoadFSHonoursPaths(t *testing.T) {
	mem := fsx.Mem()
	writeFile(t, mem, "/root/config.yaml",
		[]byte("active_profile: test\npaths:\n  skills: custom-skills\n"))
	writeFile(t, mem, "/root/profiles/test.yaml",
		[]byte("name: t\ngateway:\n  url: u\n  token: tok\n  default_model: m\n"))
	// Skill lives under the relocated dir, NOT the default skills/.
	writeFile(t, mem, "/root/custom-skills/relocated/SKILL.md",
		[]byte("---\nname: relocated\ndescription: d\n---\nbody\n"))
	// A decoy under the default dir must be ignored.
	writeFile(t, mem, "/root/skills/decoy/SKILL.md",
		[]byte("---\nname: decoy\ndescription: d\n---\nbody\n"))

	b, err := canonical.LoadFS(mem, "/root")
	require.NoError(t, err)
	require.Len(t, b.Skills, 1)
	assert.Equal(t, "relocated", b.Skills[0].Name)
}
