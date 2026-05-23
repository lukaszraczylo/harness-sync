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

func TestDiffInvokesDryRun(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"),
		[]byte("name: home\ngateway:\n  url: u\n  token: t\n  default_model: m\nmodels:\n  - {id: m}\n"), 0o600))

	reg := adapter.NewRegistry()
	reg.Register(&detectableAdapter{name: "stub", detect: true})

	var buf bytes.Buffer
	cmd := NewDiff(reg)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--root", root})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, buf.String(), "dry-run")
}
