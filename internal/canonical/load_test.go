package canonical

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	b, err := Load("testdata/sample")
	require.NoError(t, err)

	assert.Equal(t, "home", b.Config.ActiveProfile)
	assert.Equal(t, []string{"claude-code", "crush"}, b.Config.EnabledHarnesses)

	assert.Equal(t, "home", b.Profile.Name)
	assert.Equal(t, "https://gw.lan", b.Profile.Gateway.URL)

	require.Len(t, b.MCP.Servers, 1)
	assert.Equal(t, "filepuff", b.MCP.Servers[0].Name)

	require.Len(t, b.Skills, 1)
	assert.Equal(t, "hello", b.Skills[0].Name)

	require.Len(t, b.Agents, 1)
	assert.Equal(t, "sample", b.Agents[0].Name)

	require.Len(t, b.Rules, 1)
	assert.Equal(t, "sample-rule", b.Rules[0].Name)
	assert.Contains(t, b.Rules[0].Body, "Always be concise.")

	assert.Contains(t, b.Instructions.Global, "Be helpful.")
}

func TestLoadMissingProfile(t *testing.T) {
	_, err := Load("testdata/missing")
	assert.Error(t, err)
}
func TestLoadMissingConfigYAML(t *testing.T) {
	// Root exists but has no config.yaml — error must call that file out.
	// Built at runtime so we don't depend on git tracking empty dirs (it doesn't).
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o750))
	_, err := Load(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config.yaml")
}

func TestLoadMalformedConfigYAML(t *testing.T) {
	_, err := Load("testdata/malformed_config")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
	assert.Contains(t, err.Error(), "config.yaml")
}

func TestLoadMissingActiveProfile(t *testing.T) {
	// config.yaml references "nonexistent" profile which has no file.
	_, err := Load("testdata/missing_active_profile")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestLoadProfileMissingName(t *testing.T) {
	// profile YAML exists but has no 'name' field.
	_, err := Load("testdata/profile_no_name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile")
	assert.Contains(t, err.Error(), "name")
}
