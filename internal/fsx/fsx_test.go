package fsx_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/fsx"
)

func TestOSReturnsAferoOsFs(t *testing.T) {
	fs := fsx.OS()
	require.NotNil(t, fs)
	_, ok := fs.(*afero.OsFs)
	assert.True(t, ok, "OS() should return *afero.OsFs")
}

func TestMemReturnsAferoMemFs(t *testing.T) {
	fs := fsx.Mem()
	require.NotNil(t, fs)
	_, ok := fs.(*afero.MemMapFs)
	assert.True(t, ok, "Mem() should return *afero.MemMapFs")
}

func TestWriteReadRoundtrip(t *testing.T) {
	fs := fsx.Mem()
	data := []byte("hello world")
	require.NoError(t, fsx.WriteFile(fs, "/test.txt", data, 0o644))
	got, err := fsx.ReadFile(fs, "/test.txt")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}
