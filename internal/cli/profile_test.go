package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileList(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"), []byte("name: home\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "work.yaml"), []byte("name: work\n"), 0o600))

	var buf bytes.Buffer
	cmd := NewProfile()
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

	cmd := NewProfile()
	cmd.SetArgs([]string{"use", "work", "--root", root})
	require.NoError(t, cmd.Execute())

	body, err := os.ReadFile(filepath.Join(root, "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "active_profile: work")
}
