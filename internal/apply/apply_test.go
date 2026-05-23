package apply

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/gitx"
)

// failingAdapter always returns an error from Render.
type failingAdapter struct{ name, msg string }

func (f *failingAdapter) Name() string { return f.name }
func (f *failingAdapter) Detect() bool { return true }
func (f *failingAdapter) Render(_ *canonical.Bundle) (*adapter.FileSet, error) {
	return nil, errors.New(f.msg)
}
func (f *failingAdapter) Import(_ string) (*adapter.ImportResult, error) {
	return &adapter.ImportResult{}, nil
}

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

// ---------------------------------------------------------------------------
// Existing tests
// ---------------------------------------------------------------------------

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

	sp := filepath.Join(root, "state", "stub", target)
	assert.FileExists(t, sp)
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

// ---------------------------------------------------------------------------
// TestApplySymlinkFileFreshLink — SymlinkFile kind creates a symlink.
// ---------------------------------------------------------------------------

func TestApplySymlinkFileFreshLink(t *testing.T) {
	root := t.TempDir()
	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "link")
	real := filepath.Join(targetDir, "real.txt")
	require.NoError(t, os.WriteFile(real, []byte("hello\n"), 0o600))
	require.NoError(t, initCanonical(root))

	ad := &stubAdapter{
		name: "stub",
		files: []adapter.File{
			{Dest: target, Kind: adapter.SymlinkFile, SymlinkTarget: real},
		},
	}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
		// No Repo: commit path covered by TestApplyCommitsToRepo; symlink target
		// lives outside the canonical root so git-add would fail anyway.
	})
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Written)
	assert.Equal(t, 0, rep.Skipped)

	got, err := os.Readlink(target)
	require.NoError(t, err)
	assert.Equal(t, real, got)
}

// ---------------------------------------------------------------------------
// TestApplySymlinkAlreadyCorrect — skip path when symlink already points to right target.
// ---------------------------------------------------------------------------

func TestApplySymlinkAlreadyCorrect(t *testing.T) {
	root := t.TempDir()
	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "link")
	real := filepath.Join(targetDir, "real.txt")
	require.NoError(t, os.WriteFile(real, []byte("hello\n"), 0o600))
	require.NoError(t, os.Symlink(real, target))
	require.NoError(t, initCanonical(root))

	ad := &stubAdapter{
		name: "stub",
		files: []adapter.File{
			{Dest: target, Kind: adapter.SymlinkDir, SymlinkTarget: real},
		},
	}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
		Repo:     gitx.New(root),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, rep.Written)
	assert.Equal(t, 1, rep.Skipped)
}

// ---------------------------------------------------------------------------
// TestApplySymlinkReplacesExistingFile — regular file at dest gets backed up, symlink created.
// ---------------------------------------------------------------------------

func TestApplySymlinkReplacesExistingFile(t *testing.T) {
	root := t.TempDir()
	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "config.txt")
	real := filepath.Join(targetDir, "real.txt")
	require.NoError(t, os.WriteFile(real, []byte("real\n"), 0o600))
	require.NoError(t, os.WriteFile(target, []byte("old content\n"), 0o600))
	require.NoError(t, initCanonical(root))

	ad := &stubAdapter{
		name: "stub",
		files: []adapter.File{
			{Dest: target, Kind: adapter.SymlinkFile, SymlinkTarget: real},
		},
	}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
		Repo:     gitx.New(root),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Written)

	// target is now a symlink.
	got, err := os.Readlink(target)
	require.NoError(t, err)
	assert.Equal(t, real, got)

	// backup exists under <root>/backups/stub/.
	backupDir := filepath.Join(root, "backups", "stub")
	entries, err := os.ReadDir(backupDir)
	require.NoError(t, err)
	assert.NotEmpty(t, entries)
}

// ---------------------------------------------------------------------------
// TestApplyForceOverwritesDivergence — Force=true overwrites even when target diverged.
// ---------------------------------------------------------------------------

func TestApplyForceOverwritesDivergence(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "out.txt")
	require.NoError(t, initCanonical(root))
	repo := gitx.New(root)

	// First apply — writes base.
	ad1 := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("base\n")},
	}}
	_, err := Run(Options{Bundle: &canonical.Bundle{Root: root}, Adapters: []adapter.Adapter{ad1}, Repo: repo})
	require.NoError(t, err)

	// User diverges target.
	require.NoError(t, os.WriteFile(target, []byte("user-edit\n"), 0o600))

	// Second apply with Force=true — must overwrite regardless.
	ad2 := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("forced\n")},
	}}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad2},
		Repo:     repo,
		Force:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Written)
	assert.Equal(t, 0, rep.Conflicts)

	body, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "forced\n", string(body))
}

// ---------------------------------------------------------------------------
// TestApplyDryRunDoesNotWrite — DryRun=true populates Actions but writes nothing.
// ---------------------------------------------------------------------------

func TestApplyDryRunDoesNotWrite(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "out.txt")
	require.NoError(t, initCanonical(root))

	ad := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("hello\n")},
	}}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
		DryRun:   true,
	})
	require.NoError(t, err)
	// DryRun: Written counter not incremented; action recorded.
	assert.Equal(t, 0, rep.Written)
	assert.NotEmpty(t, rep.Actions)
	assert.Equal(t, "wrote", rep.Actions[0].Kind)

	// File must not exist on disk.
	_, statErr := os.Stat(target)
	assert.True(t, os.IsNotExist(statErr), "file must not be created in dry-run")
}

// ---------------------------------------------------------------------------
// TestApplyCleanThreeWayMerge — disjoint edits merge without conflict.
// ---------------------------------------------------------------------------

func TestApplyCleanThreeWayMerge(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "merge.txt")
	require.NoError(t, initCanonical(root))
	repo := gitx.New(root)

	// First apply writes base content.
	ad1 := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("line1\nline2\nline3\n")},
	}}
	_, err := Run(Options{Bundle: &canonical.Bundle{Root: root}, Adapters: []adapter.Adapter{ad1}, Repo: repo})
	require.NoError(t, err)

	// User edits line3 → "line3-user" (theirs).
	require.NoError(t, os.WriteFile(target, []byte("line1\nline2\nline3-user\n"), 0o600))

	// Adapter changes line1 → "line1-ours" (disjoint from user's line3 change).
	ad2 := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("line1-ours\nline2\nline3\n")},
	}}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad2},
		Repo:     repo,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, rep.Conflicts)
	assert.Equal(t, 1, rep.Written)

	body, err := os.ReadFile(target)
	require.NoError(t, err)
	// Merged result must contain both changes.
	assert.Contains(t, string(body), "line1-ours")
	assert.Contains(t, string(body), "line3-user")
}

// ---------------------------------------------------------------------------
// TestApplyDryRunConflict — DryRun=true on a conflict records action, no .rej written.
// ---------------------------------------------------------------------------

func TestApplyDryRunConflict(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "conflict.txt")
	require.NoError(t, initCanonical(root))
	repo := gitx.New(root)

	// First apply: write base.
	ad1 := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("base\n")},
	}}
	_, err := Run(Options{Bundle: &canonical.Bundle{Root: root}, Adapters: []adapter.Adapter{ad1}, Repo: repo})
	require.NoError(t, err)

	// User diverges.
	require.NoError(t, os.WriteFile(target, []byte("user-edit\n"), 0o600))

	// DryRun second apply with conflicting content.
	ad2 := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("new-ours\n")},
	}}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad2},
		DryRun:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Conflicts)
	// .rej must NOT exist in dry-run.
	_, statErr := os.Stat(target + ".rej")
	assert.True(t, os.IsNotExist(statErr), ".rej must not be written in dry-run")
}

// ---------------------------------------------------------------------------
// TestApplyDryRunSymlink — DryRun=true on symlink records action, no link created.
// ---------------------------------------------------------------------------

func TestApplyDryRunSymlink(t *testing.T) {
	root := t.TempDir()
	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "dry-link")
	real := filepath.Join(targetDir, "real.txt")
	require.NoError(t, os.WriteFile(real, []byte("real\n"), 0o600))
	require.NoError(t, initCanonical(root))

	ad := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.SymlinkFile, SymlinkTarget: real},
	}}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
		DryRun:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, rep.Written)
	assert.NotEmpty(t, rep.Actions)
	assert.Equal(t, "symlinked", rep.Actions[0].Kind)

	// Symlink must NOT be created.
	_, statErr := os.Lstat(target)
	assert.True(t, os.IsNotExist(statErr), "symlink must not be created in dry-run")
}

// ---------------------------------------------------------------------------
// TestApplyDryRunMerge — DryRun=true on a clean three-way merge records action, no write.
// ---------------------------------------------------------------------------

func TestApplyDryRunMerge(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "dry-merge.txt")
	require.NoError(t, initCanonical(root))
	repo := gitx.New(root)

	// Write base.
	ad1 := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("line1\nline2\nline3\n")},
	}}
	_, err := Run(Options{Bundle: &canonical.Bundle{Root: root}, Adapters: []adapter.Adapter{ad1}, Repo: repo})
	require.NoError(t, err)

	// User edits line3 (theirs).
	require.NoError(t, os.WriteFile(target, []byte("line1\nline2\nline3-user\n"), 0o600))

	// DryRun apply with disjoint change (ours edits line1).
	ad2 := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("line1-ours\nline2\nline3\n")},
	}}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad2},
		DryRun:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, rep.Written)
	assert.Equal(t, 0, rep.Conflicts)
	assert.NotEmpty(t, rep.Actions)
	assert.Equal(t, "wrote", rep.Actions[0].Kind)

	// File on disk must be unchanged (still the user's edit).
	body, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2\nline3-user\n", string(body))
}

// ---------------------------------------------------------------------------
// TestApplyDryRunSkippedWritesState — skipped path calls writeState even in dry-run? No — writeState no-ops.
// TestApplyRenderedSkippedAlreadyInSync — target already matches rendered content → skipped.
// ---------------------------------------------------------------------------

func TestApplyRenderedSkippedAlreadyInSync(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "sync.txt")
	require.NoError(t, initCanonical(root))
	repo := gitx.New(root)

	content := []byte("synced\n")

	// Write target directly (simulates already-synced state).
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o750))
	require.NoError(t, os.WriteFile(target, content, 0o600))

	ad := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: content},
	}}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
		Repo:     repo,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, rep.Written)
	assert.Equal(t, 1, rep.Skipped)
	assert.Equal(t, "skipped", rep.Actions[0].Kind)
}

// ---------------------------------------------------------------------------
// TestApplyRenderError — adapter Render error propagates.
// ---------------------------------------------------------------------------

func TestApplyRenderError(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, initCanonical(root))

	ad := &failingAdapter{name: "bad", msg: "render failed"}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render failed")
	assert.Equal(t, 0, rep.Written)
}

// ---------------------------------------------------------------------------
// TestApplyEmptyAdapterList — no adapters → empty report, no error.
// ---------------------------------------------------------------------------

func TestApplyEmptyAdapterList(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, initCanonical(root))

	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, rep.Written)
	assert.Equal(t, 0, rep.Skipped)
	assert.Equal(t, 0, rep.Conflicts)
	assert.Empty(t, rep.Actions)
}

// ---------------------------------------------------------------------------
// TestApplyCommitsToRepo — Run commits to the canonical repo after writes.
// ---------------------------------------------------------------------------

func TestApplyCommitsToRepo(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "committed.txt")
	require.NoError(t, initCanonical(root))
	repo := gitx.New(root)

	ad := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: []byte("committed\n")},
	}}
	rep, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
		Repo:     repo,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, rep.Written)

	// git log should have at least 2 commits (init + apply).
	out, gitErr := runGit(root, "log", "--oneline")
	require.NoError(t, gitErr)
	assert.GreaterOrEqual(t, countLines(out), 2, "expected at least 2 commits in git log")
}

// ---------------------------------------------------------------------------
// TestApplyStateRoundTrip — re-apply same content → skipped (no second write).
// ---------------------------------------------------------------------------

func TestApplyStateRoundTrip(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "round.txt")
	require.NoError(t, initCanonical(root))
	repo := gitx.New(root)

	content := []byte("stable\n")
	ad := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.RenderedFile, Content: content},
	}}
	opts := Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
		Repo:     repo,
	}

	// First apply: writes.
	rep1, err := Run(opts)
	require.NoError(t, err)
	assert.Equal(t, 1, rep1.Written)

	// Second apply: target matches content → skipped.
	rep2, err := Run(opts)
	require.NoError(t, err)
	assert.Equal(t, 0, rep2.Written)
	assert.Equal(t, 1, rep2.Skipped)
}

// ---------------------------------------------------------------------------
// TestApplyUnknownFileKind — unknown Kind returns error.
// ---------------------------------------------------------------------------

func TestApplyUnknownFileKind(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "unknown.txt")
	require.NoError(t, initCanonical(root))

	ad := &stubAdapter{name: "stub", files: []adapter.File{
		{Dest: target, Kind: adapter.Kind(99)},
	}}
	_, err := Run(Options{
		Bundle:   &canonical.Bundle{Root: root},
		Adapters: []adapter.Adapter{ad},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown file kind")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

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

func runGit(dir string, args ...string) (string, error) {
	allArgs := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", allArgs...).Output() //nolint:gosec
	return string(out), err
}

func countLines(s string) int {
	n := 0
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}
