package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateDryRunPrintsCommand(t *testing.T) {
	cmd := NewUpdate()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--dry-run", "--install-dir", "/tmp/test-dir"})
	require.NoError(t, cmd.Execute())

	out := buf.String()
	assert.Contains(t, out, "Installing latest harness-sync into /tmp/test-dir")
	assert.Contains(t, out, "curl -fsSL")
	assert.Contains(t, out, "install.sh")
	assert.Contains(t, out, "INSTALL_DIR=")
	// Should NOT have run anything
	assert.False(t, strings.Contains(out, "[INFO]"), "dry run must not invoke install.sh")
}

func TestUpdateResolvesCurrentBinaryDirByDefault(t *testing.T) {
	cmd := NewUpdate()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--dry-run"})
	require.NoError(t, cmd.Execute())

	// Should mention SOME absolute path (the test binary's directory)
	out := buf.String()
	assert.Contains(t, out, "Installing latest harness-sync into /")
}
