package common_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
)

// ReadExistingFile: missing file returns (nil, nil). Permission errors and
// other I/O failures must NOT be silently swallowed — those would let
// read-modify-write code clobber the user's on-disk state.

func TestReadExistingFileMissing(t *testing.T) {
	got, err := common.ReadExistingFile(filepath.Join(t.TempDir(), "absent"))
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestReadExistingFileContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.txt")
	require.NoError(t, os.WriteFile(path, []byte("body\n"), 0o600))

	got, err := common.ReadExistingFile(path)
	require.NoError(t, err)
	assert.Equal(t, []byte("body\n"), got)
}

func TestReadExistingFilePropagatesError(t *testing.T) {
	// Create a directory the OS will refuse us read access to.
	if os.Geteuid() == 0 {
		t.Skip("running as root; cannot test permission denial")
	}
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "locked"), 0o000))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "locked"), 0o755) })

	_, err := common.ReadExistingFile(filepath.Join(dir, "locked", "file"))
	require.Error(t, err)
	assert.False(t, os.IsNotExist(err), "permission denied must not be reported as not-exist")
	assert.True(t, errors.Is(err, os.ErrPermission) || isPermission(err),
		"expected permission error, got %v", err)
}

func isPermission(err error) bool {
	// Some platforms wrap the error in *os.PathError. Check the string too
	// for portability — it's a coarse signal but adequate here.
	return err != nil && (contains(err.Error(), "permission") || contains(err.Error(), "access denied"))
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
