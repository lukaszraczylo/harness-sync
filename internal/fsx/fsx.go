// Package fsx provides a small filesystem abstraction so file-touching code
// can be tested against an in-memory filesystem.
package fsx

import (
	"os"

	"github.com/spf13/afero"
)

// FS is the harness-sync filesystem interface — alias for afero.Fs so the
// dependency stays in one place.
type FS = afero.Fs

// OS returns the production OS-backed filesystem.
func OS() FS { return afero.NewOsFs() }

// Mem returns an empty in-memory filesystem for tests.
func Mem() FS { return afero.NewMemMapFs() }

// WriteFile writes data to path on fs.
func WriteFile(fs FS, path string, data []byte, perm os.FileMode) error {
	return afero.WriteFile(fs, path, data, perm)
}

// ReadFile reads the named file from fs and returns its contents.
func ReadFile(fs FS, path string) ([]byte, error) {
	return afero.ReadFile(fs, path)
}
