package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

type importableAdapter struct {
	res  *adapter.ImportResult
	name string
}

func (i *importableAdapter) Name() string { return i.name }
func (i *importableAdapter) Detect() bool { return true }
func (i *importableAdapter) Render(_ *canonical.Bundle) (*adapter.FileSet, error) {
	return adapter.NewFileSet(), nil
}
func (i *importableAdapter) Import(_ string) (*adapter.ImportResult, error) { return i.res, nil }

func TestInitImportWritesCanonical(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	reg := adapter.NewRegistry()
	reg.Register(&importableAdapter{
		name: "stub",
		res: &adapter.ImportResult{
			Skills:       []canonical.Skill{{Name: "hi", Body: "---\nname: hi\n---\nhi body", Path: "hi/SKILL.md"}},
			Instructions: "# global",
			MCP:          []canonical.MCPServer{{Name: "filepuff", Command: "/bin/x"}},
		},
	})

	cmd := NewInit(reg)
	cmd.SetArgs([]string{"--root", root, "--home", home, "--from", "stub", "--no-prompt"})
	require.NoError(t, cmd.Execute())

	assert.FileExists(t, filepath.Join(root, "config.yaml"))
	assert.FileExists(t, filepath.Join(root, "instructions", "global.md"))
	assert.FileExists(t, filepath.Join(root, "mcp.yaml"))
	assert.FileExists(t, filepath.Join(root, "skills", "hi", "SKILL.md"))
	_, err := os.Stat(filepath.Join(root, ".git"))
	require.NoError(t, err)
}

func makeStubReg() *adapter.Registry {
	reg := adapter.NewRegistry()
	reg.Register(&importableAdapter{
		name: "stub",
		res:  &adapter.ImportResult{},
	})
	return reg
}

// TestInitRefusesOverwriteWithoutForce: second init without --force must error.
func TestInitRefusesOverwriteWithoutForce(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	// First init — should succeed.
	cmd := NewInit(makeStubReg())
	cmd.SetArgs([]string{"--root", root, "--home", home, "--from", "stub", "--no-prompt"})
	require.NoError(t, cmd.Execute())

	// Second init — must fail without --force.
	cmd2 := NewInit(makeStubReg())
	cmd2.SetArgs([]string{"--root", root, "--home", home, "--from", "stub", "--no-prompt"})
	err := cmd2.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already initialised")
	assert.Contains(t, err.Error(), "--force")
}

// TestInitForcePreservesExistingProfile: --force re-imports but does NOT overwrite imported.yaml.
func TestInitForcePreservesExistingProfile(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	// First init writes placeholder profile.
	cmd := NewInit(makeStubReg())
	cmd.SetArgs([]string{"--root", root, "--home", home, "--from", "stub", "--no-prompt"})
	require.NoError(t, cmd.Execute())

	// User edits the profile.
	profPath := filepath.Join(root, "profiles", "imported.yaml")
	userEdit := []byte("name: imported\ngateway:\n  url: \"https://my.gateway\"\n  token: \"tok\"\n  default_model: \"m\"\nmodels: []\n")
	require.NoError(t, os.WriteFile(profPath, userEdit, 0o600))

	// Re-init with --force.
	cmd2 := NewInit(makeStubReg())
	cmd2.SetArgs([]string{"--root", root, "--home", home, "--from", "stub", "--no-prompt", "--force"})
	require.NoError(t, cmd2.Execute())

	// User edits must be preserved.
	got, err := os.ReadFile(profPath)
	require.NoError(t, err)
	assert.Equal(t, string(userEdit), string(got))
}
