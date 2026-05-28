package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/gitx"
)

// fileAdapter renders a single file whose content is the profile default model,
// so a rollback round-trip is observable on disk.
type fileAdapter struct {
	name string
	dest string
}

func (f *fileAdapter) Name() string { return f.name }
func (f *fileAdapter) Detect() bool { return true }
func (f *fileAdapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	fs.Add(adapter.File{Dest: f.dest, Kind: adapter.RenderedFile, Content: []byte(b.Profile.Gateway.DefaultModel + "\n")})
	return fs, nil
}
func (f *fileAdapter) Import(_ string) (*adapter.ImportResult, error) {
	return &adapter.ImportResult{}, nil
}
func (f *fileAdapter) Capabilities() adapter.HarnessCapabilities {
	return adapter.HarnessCapabilities{ManagesProviders: true}
}

func TestRollbackHelp(t *testing.T) {
	cmd := NewRollback(adapter.NewRegistry())
	assert.Equal(t, "rollback [n]", cmd.Use)
}

func TestRollbackRejectsBadArg(t *testing.T) {
	cmd := NewRollback(adapter.NewRegistry())
	cmd.SetArgs([]string{"abc"})
	err := cmd.Execute()
	require.Error(t, err, "non-integer arg must error (non-zero exit), not print help and succeed")
}

func TestRollbackRevertsAndReappliesToHarness(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	writeProfile := func(model string) {
		require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"),
			[]byte("name: home\ngateway:\n  url: u\n  token: t\n  default_model: "+model+"\nmodels:\n  - {id: "+model+"}\n"), 0o600))
	}

	repo := gitx.New(root)
	require.NoError(t, repo.Init())
	require.NoError(t, repo.Configure("t", "t@t"))

	target := filepath.Join(t.TempDir(), "out.txt")
	reg := adapter.NewRegistry()
	reg.Register(&fileAdapter{name: "stub", dest: target})

	doApply := func() {
		var buf bytes.Buffer
		cmd := NewApply(reg)
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{"--root", root, "--yes"})
		require.NoError(t, cmd.Execute())
	}

	writeProfile("v1")
	doApply()
	got, _ := os.ReadFile(target)
	require.Equal(t, "v1\n", string(got))

	writeProfile("v2")
	doApply()
	got, _ = os.ReadFile(target)
	require.Equal(t, "v2\n", string(got))

	// rollback 1 reverts the v2 commit AND re-applies, so the harness file goes
	// back to v1 — a bare git revert would have left it at v2.
	var buf bytes.Buffer
	cmd := NewRollback(reg)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--root", root})
	require.NoError(t, cmd.Execute())

	got, _ = os.ReadFile(target)
	assert.Equal(t, "v1\n", string(got), "rollback must propagate reverted state to the harness file")
}
