package canonical_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/fsx"
)

// writeFile is a convenience helper: writes bytes to path on fs, creating any
// parent directories first.
func writeFile(t *testing.T, fs fsx.FS, path string, content []byte) {
	t.Helper()
	require.NoError(t, fs.MkdirAll(parentDir(path), 0o755))
	require.NoError(t, afero.WriteFile(fs, path, content, 0o644))
}

func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "/"
}

func TestLoadFSFromMemFS(t *testing.T) {
	mem := fsx.Mem()

	configYAML := []byte("active_profile: test\n")
	profileYAML := []byte("name: Test Profile\ngateway:\n  url: https://example.com\n  token: tok\n  default_model: gpt4\n")
	skillBody := []byte("---\nname: my-skill\ndescription: A test skill\n---\n# My Skill\nsome content\n")

	writeFile(t, mem, "/root/config.yaml", configYAML)
	writeFile(t, mem, "/root/profiles/test.yaml", profileYAML)
	writeFile(t, mem, "/root/skills/my-skill/SKILL.md", skillBody)

	bundle, err := canonical.LoadFS(mem, "/root")
	require.NoError(t, err)
	require.NotNil(t, bundle)

	assert.Equal(t, "test", bundle.Config.ActiveProfile)
	assert.Equal(t, "Test Profile", bundle.Profile.Name)
	require.Len(t, bundle.Skills, 1)
	assert.Equal(t, "my-skill", bundle.Skills[0].Name)
	assert.Equal(t, "A test skill", bundle.Skills[0].Description)
}

func TestLoadFSEmptyProfile(t *testing.T) {
	mem := fsx.Mem()

	// config.yaml present but profile file is missing
	writeFile(t, mem, "/root/config.yaml", []byte("active_profile: missing\n"))

	_, err := canonical.LoadFS(mem, "/root")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestLoadFSMissingRoot(t *testing.T) {
	mem := fsx.Mem()

	_, err := canonical.LoadFS(mem, "/does-not-exist")
	require.Error(t, err)
}

func TestLoadFSActiveProfileRequired(t *testing.T) {
	mem := fsx.Mem()

	writeFile(t, mem, "/root/config.yaml", []byte("enabled_harnesses: []\n"))

	_, err := canonical.LoadFS(mem, "/root")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active_profile is required")
}
