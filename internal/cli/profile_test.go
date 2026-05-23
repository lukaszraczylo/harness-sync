package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func TestProfileList(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"), []byte("name: home\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "work.yaml"), []byte("name: work\n"), 0o600))

	var buf bytes.Buffer
	cmd := NewProfile(nil)
	cmd.SetArgs([]string{"list", "--root", root})
	cmd.SetOut(&buf)
	require.NoError(t, cmd.Execute())

	out := buf.String()
	assert.Contains(t, out, "home")
	assert.Contains(t, out, "work")
}

func TestProfileUse(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "work.yaml"), []byte("name: work\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o600))

	cmd := NewProfile(nil)
	cmd.SetArgs([]string{"use", "work", "--root", root})
	require.NoError(t, cmd.Execute())

	body, err := os.ReadFile(filepath.Join(root, "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "active_profile: work")
}

// TestProfileUseHintMessage verifies the "run apply" hint is printed.
func TestProfileUseHintMessage(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "work.yaml"), []byte("name: work\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o600))

	var buf bytes.Buffer
	cmd := NewProfile(nil)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"use", "work", "--root", root})
	require.NoError(t, cmd.Execute())

	assert.Contains(t, buf.String(), "Switched active profile to")
	assert.Contains(t, buf.String(), "harness-sync apply")
}

func TestProfileRename(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "old.yaml"), []byte("name: old\ngateway:\n  url: https://gw\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: other\n"), 0o600))

	var buf bytes.Buffer
	cmd := NewProfile(nil)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"rename", "old", "new", "--root", root})
	require.NoError(t, cmd.Execute())

	// old file gone, new file present with updated name field.
	_, err := os.Stat(filepath.Join(root, "profiles", "old.yaml"))
	assert.True(t, os.IsNotExist(err), "old.yaml should be removed")
	body, err := os.ReadFile(filepath.Join(root, "profiles", "new.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "name: new")
	assert.NotContains(t, string(body), "name: old")

	// config.yaml not modified — old was not active.
	cfg, err := os.ReadFile(filepath.Join(root, "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(cfg), "active_profile: other")

	assert.Contains(t, buf.String(), `"old" → "new"`)
}

func TestProfileRenameUpdatesActiveProfile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "old.yaml"), []byte("name: old\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: old\n"), 0o600))

	cmd := NewProfile(nil)
	cmd.SetArgs([]string{"rename", "old", "new", "--root", root})
	require.NoError(t, cmd.Execute())

	cfg, err := os.ReadFile(filepath.Join(root, "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(cfg), "active_profile: new")
}

func TestProfileRenameErrorNotFound(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))

	cmd := NewProfile(nil)
	cmd.SetArgs([]string{"rename", "missing", "new", "--root", root})
	assert.Error(t, cmd.Execute())
}

func TestProfileRenameErrorAlreadyExists(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "old.yaml"), []byte("name: old\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "new.yaml"), []byte("name: new\n"), 0o600))

	cmd := NewProfile(nil)
	cmd.SetArgs([]string{"rename", "old", "new", "--root", root})
	assert.Error(t, cmd.Execute())
}

// TestProfileUseWithApply verifies --apply triggers apply after switching.
func TestProfileUseWithApply(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	// Profile must be complete so apply doesn't refuse.
	profYAML := "name: work\ngateway:\n  url: https://gw\n  token: tok\n  default_model: m\nmodels:\n  - {id: m}\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "work.yaml"), []byte(profYAML), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o600))

	reg := adapter.NewRegistry()
	// No adapters detected → apply runs but writes 0 files (nothing to render).
	reg.Register(&detectableAdapter{name: "stub", detect: false})

	var buf bytes.Buffer
	cmd := NewProfile(reg)
	cmd.SetOut(&buf)
	// --yes bypasses first-run prompt in apply; state/ won't exist.
	cmd.SetArgs([]string{"use", "work", "--root", root, "--apply"})
	require.NoError(t, cmd.Execute())

	out := buf.String()
	assert.Contains(t, out, "Switched active profile")
	// apply output: "applied: 0 written ..."
	assert.True(t, strings.Contains(out, "applied") || strings.Contains(out, "dry-run"),
		"expected apply output in: %s", out)

	// config.yaml must now reference work.
	body, err := os.ReadFile(filepath.Join(root, "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "active_profile: work")
}
