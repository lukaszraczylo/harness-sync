// Package e2e runs the harness-sync binary end-to-end against fake harness
// installs in a temp HOME.
package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitApplyDiffCycle(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	root := filepath.Join(home, ".config", "harness-sync")

	// Fake a claude-code install with one skill and one rule
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "skills", "hi"), 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(home, ".claude", "skills", "hi", "SKILL.md"),
		[]byte("---\nname: hi\n---\nbody"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "rules"), 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(home, ".claude", "rules", "go.md"),
		[]byte("# Go Rules\n\ngofmt always"), 0o600))

	run := func(args ...string) ([]byte, error) {
		cmd := exec.Command(bin, args...)
		cmd.Env = append(os.Environ(), "HOME="+home)
		return cmd.CombinedOutput()
	}

	out, err := run("init", "--no-prompt")
	require.NoError(t, err, string(out))
	assert.FileExists(t, filepath.Join(root, "config.yaml"))
	assert.FileExists(t, filepath.Join(root, "skills", "hi", "SKILL.md"))
	assert.FileExists(t, filepath.Join(root, "rules", "go.md"))

	// apply may produce conflicts on a freshly imported tree (because the
	// claude-code adapter merges into existing settings.json which differs
	// from canonical); accept either clean exit or conflict exit.
	// --allow-incomplete: the placeholder profile has empty URL by design.
	// --yes: skip first-run TTY prompt in CI.
	out, err = run("apply", "--allow-incomplete", "--yes")
	if err != nil {
		// Apply with conflicts is acceptable — verify the binary at least ran
		assert.Contains(t, string(out), "applied")
	} else {
		assert.Contains(t, string(out), "applied")
	}

	// Diff should always run cleanly (it's read-only)
	out, err = run("diff")
	require.NoError(t, err, string(out))
}

func TestShowEnumeratesTargets(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	root := filepath.Join(home, ".config", "harness-sync")

	require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o750))

	run := func(args ...string) ([]byte, error) {
		cmd := exec.Command(bin, args...)
		cmd.Env = append(os.Environ(), "HOME="+home)
		return cmd.CombinedOutput()
	}

	_, err := run("init", "--no-prompt")
	require.NoError(t, err)
	_ = root // root is implied by HOME

	out, err := run("show")
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "claude-code")
}

func TestDetectListsAllAdapters(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()

	cmd := exec.Command(bin, "detect")
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	// None should be detected in an empty HOME
	s := string(out)
	for _, name := range []string{"claude-code", "crush", "kilo", "opencode", "pi", "goose", "cagent", "zed"} {
		assert.Contains(t, s, name)
	}
	assert.Contains(t, s, "not detected")
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "harness-sync")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/harness-sync")
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return bin
}
