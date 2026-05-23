package canonical

import (
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

	assert.Contains(t, b.Instructions.Global, "Be helpful.")
}

func TestLoadMissingProfile(t *testing.T) {
	_, err := Load("testdata/missing")
	assert.Error(t, err)
}
