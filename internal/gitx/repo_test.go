package gitx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitAddCommit(t *testing.T) {
	dir := t.TempDir()
	r := New(dir)

	require.NoError(t, r.Init())
	require.NoError(t, r.Configure("nobody", "nobody@example.com"))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o600))
	require.NoError(t, r.AddAll())
	require.NoError(t, r.Commit("initial"))

	head, err := r.HeadCommit()
	require.NoError(t, err)
	assert.NotEmpty(t, head)
}

func TestShowFileAtHead(t *testing.T) {
	dir := t.TempDir()
	r := New(dir)
	require.NoError(t, r.Init())
	require.NoError(t, r.Configure("nobody", "nobody@example.com"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("v1"), 0o600))
	require.NoError(t, r.AddAll())
	require.NoError(t, r.Commit("v1"))

	body, err := r.ShowFileAtHead("a.txt")
	require.NoError(t, err)
	assert.Equal(t, "v1", string(body))

	_, err = r.ShowFileAtHead("nope.txt")
	assert.Error(t, err)
}

func TestHasStagedChangesEmptyAfterInit(t *testing.T) {
	dir := t.TempDir()
	r := New(dir)
	require.NoError(t, r.Init())
	require.NoError(t, r.Configure("nobody", "nobody@example.com"))
	// No commits yet, working tree clean.
	dirty, err := r.HasStagedChanges()
	require.NoError(t, err)
	assert.False(t, dirty)
}

func TestHasStagedChangesAfterAdd(t *testing.T) {
	dir := t.TempDir()
	r := New(dir)
	require.NoError(t, r.Init())
	require.NoError(t, r.Configure("nobody", "nobody@example.com"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o600))
	require.NoError(t, r.AddAll())

	dirty, err := r.HasStagedChanges()
	require.NoError(t, err)
	assert.True(t, dirty)

	// After commit, index matches HEAD again → no staged changes.
	require.NoError(t, r.Commit("c"))
	dirty, err = r.HasStagedChanges()
	require.NoError(t, err)
	assert.False(t, dirty)
}
