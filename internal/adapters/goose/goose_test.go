package goose

import (
	"encoding/json"
	"os"
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
			Gateway: canonical.Gateway{DefaultModel: "anthropic/claude-sonnet-4-6"},
		},
		MCP: canonical.MCPRegistry{Servers: []canonical.MCPServer{
			{Name: "filepuff", Command: "/bin/filepuff", Args: []string{"--serve"}},
		}},
	}
}

func TestGooseRenderProducesExpectedTargets(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	fs, err := ad.Render(bundle())
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cfgDest := filepath.Join(home, ".config", "goose", "config.yaml")
	require.Contains(t, seen, cfgDest)
	assert.Equal(t, adapter.RenderedFile, seen[cfgDest].Kind)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(seen[cfgDest].Content, &parsed))
	assert.Equal(t, "anthropic", parsed["GOOSE_PROVIDER"])
	assert.Equal(t, "claude-sonnet-4-6", parsed["GOOSE_MODEL"])
	assert.Contains(t, parsed, "extensions")

	exts := parsed["extensions"].(map[string]any)
	fp := exts["filepuff"].(map[string]any)
	assert.Equal(t, "stdio", fp["type"])
	assert.Equal(t, "/bin/filepuff", fp["cmd"])
}

func TestGooseRenderMergesExistingKeys(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "goose")
	require.NoError(t, os.MkdirAll(base, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(base, "config.yaml"),
		[]byte("GOOSE_TEMPERATURE: \"0.7\"\nGOOSE_PROVIDER: old\n"), 0o600))

	ad := New(WithHome(home))
	fs, err := ad.Render(bundle())
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(seen[filepath.Join(base, "config.yaml")].Content, &parsed))
	// Existing unrelated key preserved.
	assert.Equal(t, "0.7", parsed["GOOSE_TEMPERATURE"])
	// Provider updated.
	assert.Equal(t, "anthropic", parsed["GOOSE_PROVIDER"])
}

func TestGooseRenderGatewayURL(t *testing.T) {
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

	cfgDest := filepath.Join(home, ".config", "goose", "config.yaml")
	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(seen[cfgDest].Content, &parsed))
	// Custom gateway → GOOSE_PROVIDER is the custom provider name.
	assert.Equal(t, "custom_harness-sync-gateway", parsed["GOOSE_PROVIDER"])
}

func TestGooseRenderProducesCustomProviderFile(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{URL: "https://gw", Token: "tok", DefaultModel: "sonnet"},
			Models:  []canonical.Model{{ID: "claude-sonnet-4-6"}},
		},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	cpDest := filepath.Join(home, ".config", "goose", "custom_providers", "custom_harness-sync-gateway.json")
	require.Contains(t, seen, cpDest)
	assert.Equal(t, adapter.RenderedFile, seen[cpDest].Kind)

	var cp map[string]any
	require.NoError(t, json.Unmarshal(seen[cpDest].Content, &cp))
	assert.Equal(t, "custom_harness-sync-gateway", cp["name"])
	assert.Equal(t, "openai", cp["engine"])
	assert.Equal(t, true, cp["requires_auth"])
	assert.Equal(t, "CUSTOM_HARNESS_SYNC_GATEWAY_API_KEY", cp["api_key_env"])
	models, ok := cp["models"].([]any)
	require.True(t, ok)
	require.Len(t, models, 1)
	m0 := models[0].(map[string]any)
	assert.Equal(t, "claude-sonnet-4-6", m0["name"])
}

func TestGooseImport(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".config", "goose")
	require.NoError(t, os.MkdirAll(base, 0o750))

	cfg := map[string]any{
		"GOOSE_PROVIDER": "anthropic",
		"GOOSE_MODEL":    "claude-sonnet-4-6",
		"extensions": map[string]any{
			"filepuff": map[string]any{
				"type":    "stdio",
				"cmd":     "/bin/filepuff",
				"args":    []string{"--serve"},
				"enabled": true,
			},
		},
	}
	body, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(base, "config.yaml"), body, 0o600))

	ad := New(WithHome(home))
	res, err := ad.Import(home)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.MCP, 1)
	assert.Equal(t, "filepuff", res.MCP[0].Name)
	assert.Equal(t, "/bin/filepuff", res.MCP[0].Command)
}

func TestGooseImportMissing(t *testing.T) {
	ad := New(WithHome(t.TempDir()))
	res, err := ad.Import(t.TempDir())
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Empty(t, res.MCP)
}

func TestSplitProviderModel(t *testing.T) {
	t.Run("with slash", func(t *testing.T) {
		p, m := splitProviderModel("anthropic/claude-sonnet-4-6")
		assert.Equal(t, "anthropic", p)
		assert.Equal(t, "claude-sonnet-4-6", m)
	})
	t.Run("no slash", func(t *testing.T) {
		p, m := splitProviderModel("claude-sonnet")
		assert.Equal(t, "anthropic", p)
		assert.Equal(t, "claude-sonnet", m)
	})
}
