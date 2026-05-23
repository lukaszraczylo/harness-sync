package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func TestApplyDryRun(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"),
		[]byte("name: home\ngateway:\n  url: u\n  token: t\n  default_model: m\nmodels:\n  - {id: m}\n"), 0o600))

	reg := adapter.NewRegistry()
	reg.Register(&detectableAdapter{name: "stub", detect: true})

	var buf bytes.Buffer
	cmd := NewApply(reg)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--dry-run", "--root", root})
	require.NoError(t, cmd.Execute())

	assert.Contains(t, buf.String(), "dry-run")
}

// TestApplyRefusesIncompleteProfile: apply must error when gateway.url or
// gateway.default_model is empty.
func TestApplyRefusesIncompleteProfile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: bad\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	// Profile with empty url and default_model.
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "bad.yaml"),
		[]byte("name: bad\ngateway:\n  url: \"\"\n  token: dummy\n  default_model: \"\"\nmodels: []\n"), 0o600))

	reg := adapter.NewRegistry()
	reg.Register(&detectableAdapter{name: "stub", detect: true})

	cmd := NewApply(reg)
	cmd.SetArgs([]string{"--root", root})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incomplete")
}

// TestApplyAllowIncompleteFlag: --allow-incomplete bypasses the gateway check.
func TestApplyAllowIncompleteFlag(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: bad\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "bad.yaml"),
		[]byte("name: bad\ngateway:\n  url: \"\"\n  token: dummy\n  default_model: \"\"\nmodels: []\n"), 0o600))

	reg := adapter.NewRegistry()
	reg.Register(&detectableAdapter{name: "stub", detect: true})

	var buf bytes.Buffer
	cmd := NewApply(reg)
	cmd.SetOut(&buf)
	// --yes skips first-run prompt; --allow-incomplete skips gateway check.
	cmd.SetArgs([]string{"--root", root, "--dry-run", "--allow-incomplete", "--yes"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, buf.String(), "dry-run")
}
