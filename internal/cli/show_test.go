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
)

type renderingAdapter struct {
	name   string
	files  []adapter.File
	detect bool
}

func (r *renderingAdapter) Name() string { return r.name }
func (r *renderingAdapter) Detect() bool { return r.detect }
func (r *renderingAdapter) Import(_ string) (*adapter.ImportResult, error) {
	return &adapter.ImportResult{}, nil
}
func (r *renderingAdapter) Render(_ *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	for _, f := range r.files {
		fs.Add(f)
	}
	return fs, nil
}

func TestShowPrintsTargets(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"),
		[]byte("name: home\ngateway:\n  url: u\n  token: t\n  default_model: m\nmodels:\n  - {id: m}\n"), 0o600))

	reg := adapter.NewRegistry()
	reg.Register(&renderingAdapter{
		name:   "stub",
		detect: true,
		files: []adapter.File{
			{Dest: "/tmp/out.json", Kind: adapter.RenderedFile, Content: []byte("{}\n")},
			{Dest: "/tmp/link", Kind: adapter.SymlinkDir, SymlinkTarget: "/canon/skills"},
		},
	})

	var buf bytes.Buffer
	cmd := NewShow(reg)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--root", root})
	require.NoError(t, cmd.Execute())

	out := buf.String()
	assert.Contains(t, out, "stub")
	assert.Contains(t, out, "rendered")
	assert.Contains(t, out, "symlink-dir")
	assert.Contains(t, out, "/tmp/out.json")
	assert.Contains(t, out, "/canon/skills")
}

func TestShowAllIncludesUndetected(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"),
		[]byte("name: home\ngateway:\n  url: u\n  token: t\n  default_model: m\nmodels:\n  - {id: m}\n"), 0o600))

	reg := adapter.NewRegistry()
	reg.Register(&renderingAdapter{name: "absent", detect: false})

	var buf bytes.Buffer
	cmd := NewShow(reg)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--root", root, "--all"})
	require.NoError(t, cmd.Execute())

	assert.Contains(t, buf.String(), "absent")
	assert.Contains(t, buf.String(), "not detected")
}
