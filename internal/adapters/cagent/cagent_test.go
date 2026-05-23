package cagent

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func bundle() *canonical.Bundle {
	return &canonical.Bundle{
		Root: "/canon",
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{
				URL:          "https://gw",
				DefaultModel: "claude-sonnet-4-6",
			},
		},
		Instructions: canonical.Instructions{Global: "# global instruction"},
		MCP: canonical.MCPRegistry{Servers: []canonical.MCPServer{
			{Name: "filepuff", Command: "/bin/filepuff", Args: []string{"--serve"}},
		}},
	}
}

func TestCagentRenderProducesExpectedTargets(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	fs, err := ad.Render(bundle())
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "cagent", "default.yaml")
	require.Contains(t, seen, cfgDest)
	assert.Equal(t, adapter.RenderedFile, seen[cfgDest].Kind)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(seen[cfgDest].Content, &parsed))

	assert.Equal(t, 8, parsed["version"])

	agents := parsed["agents"].(map[string]any)
	def := agents["default"].(map[string]any)
	assert.Equal(t, "hs-gw/claude-sonnet-4-6", def["model"])
	assert.Contains(t, def["instruction"], "# global instruction")

	// providers map (new schema).
	providers := parsed["providers"].(map[string]any)
	gwProv := providers["hs-gw"].(map[string]any)
	assert.Equal(t, "https://gw", gwProv["base_url"])
	assert.Equal(t, "openai", gwProv["provider"])
	assert.NotContains(t, gwProv, "token_key")

	// models map.
	models := parsed["models"].(map[string]any)
	gw := models["hs-gw"].(map[string]any)
	assert.Equal(t, "hs-gw", gw["provider"])
	assert.Equal(t, "claude-sonnet-4-6", gw["model"])

	mcps := parsed["mcps"].(map[string]any)
	fp := mcps["filepuff"].(map[string]any)
	assert.Equal(t, "/bin/filepuff", fp["command"])
}

func TestCagentRenderProvidersAndModelsMaps(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{
				URL:          "https://gw",
				Token:        "tok",
				DefaultModel: "claude-sonnet-4-6",
			},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)
	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "cagent", "default.yaml")
	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(seen[cfgDest].Content, &parsed))

	// providers block present.
	providers, ok := parsed["providers"].(map[string]any)
	require.True(t, ok, "providers must be a map")
	gw, ok := providers["hs-gw"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://gw", gw["base_url"])
	assert.Equal(t, "tok", gw["api_key"])
	assert.Equal(t, "openai", gw["provider"])
	assert.NotContains(t, gw, "token_key")

	// agents.default.model uses inline shorthand.
	agents := parsed["agents"].(map[string]any)
	def := agents["default"].(map[string]any)
	assert.Equal(t, "hs-gw/claude-sonnet-4-6", def["model"])
}

func TestCagentRenderNoMCP(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{URL: "https://gw", DefaultModel: "sonnet"},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "cagent", "default.yaml")
	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(seen[cfgDest].Content, &parsed))
	assert.NotContains(t, parsed, "mcps")
}

func TestCagentImportReturnsEmpty(t *testing.T) {
	ad := New(WithHome(t.TempDir()))
	res, err := ad.Import(t.TempDir())
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Empty(t, res.MCP)
	assert.Empty(t, res.Skills)
	assert.Empty(t, res.Agents)
}

func TestCagentDetect(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	assert.False(t, ad.Detect())
}
