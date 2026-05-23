package apply

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/gitx"
)

type stubAdapter struct {
	name  string
	files []adapter.File
}

func (s *stubAdapter) Name() string { return s.name }
func (s *stubAdapter) Detect() bool { return true }
func (s *stubAdapter) Import(_ string) (*adapter.ImportResult, error) {
	return &adapter.ImportResult{}, nil
}
func (s *stubAdapter) Render(_ *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	for _, f := range s.files {
		fs.Add(f)
	}
	return fs, nil
}

func TestApplyRenderedFileFreshWrite(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "out.json")

	require.NoError(t, initCanonical(root))

	ad := &stubAdapter{
		name: "stub",
		files: []adapter.File{
			{Dest: target, Kind: adapter.RenderedFile, Content: []byte("{}\n")},
		},
	}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
		Repo:     gitx.New(root),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Written)
	assert.Equal(t, 0, rep.Conflicts)

	body, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "{}\n", string(body))

	statePath := filepath.Join(root, "state", "stub", target)
	assert.FileExists(t, statePath)
}

func TestApplyConflictWritesRej(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "out.txt")

	require.NoError(t, initCanonical(root))
	repo := gitx.New(root)

	ad1 := &stubAdapter{
		name: "stub",
		files: []adapter.File{
			{Dest: target, Kind: adapter.RenderedFile, Content: []byte("base\n")},
		},
	}
	_, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad1},
		Repo:     repo,
	})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(target, []byte("user-edit\n"), 0o600)) //nolint:gosec

	ad2 := &stubAdapter{
		name: "stub",
		files: []adapter.File{
			{Dest: target, Kind: adapter.RenderedFile, Content: []byte("new\n")},
		},
	}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad2},
		Repo:     repo,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Conflicts)
	assert.FileExists(t, target+".rej")
}

func initCanonical(root string) error {
	if err := os.MkdirAll(filepath.Join(root, "state"), 0o750); err != nil {
		return err
	}
	r := gitx.New(root)
	if err := r.Init(); err != nil {
		return err
	}
	if err := r.Configure("test", "test@example.com"); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, ".gitkeep"), []byte{}, 0o600); err != nil {
		return err
	}
	if err := r.AddAll(); err != nil {
		return err
	}
	return r.Commit("init")
}
