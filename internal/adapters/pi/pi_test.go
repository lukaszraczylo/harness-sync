package pi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestPiRender(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{
		Root: "/canon",
		Profile: canonical.Profile{
			Gateway: canonical.Gateway{URL: "https://gw", Token: "dummy", DefaultModel: "sonnet"},
			Models:  []canonical.Model{{ID: "claude-sonnet-4-6", Alias: "Sonnet"}},
		},
		Instructions: canonical.Instructions{Global: "# global"},
	}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	skillsDest := filepath.Join(home, ".pi", "agent", "skills")
	assert.Equal(t, adapter.SymlinkDir, seen[skillsDest].Kind)
	assert.Equal(t, "/canon/skills", seen[skillsDest].SymlinkTarget)

	agentsDest := filepath.Join(home, ".pi", "agent", "AGENTS.md")
	assert.Equal(t, adapter.RenderedFile, seen[agentsDest].Kind)
	assert.Contains(t, string(seen[agentsDest].Content), "# global")

	settingsDest := filepath.Join(home, ".pi", "agent", "settings.json")
	var settings map[string]any
	require.NoError(t, json.Unmarshal(seen[settingsDest].Content, &settings))
	assert.Equal(t, "hs-gw", settings["defaultProvider"])
	assert.Equal(t, "sonnet", settings["defaultModel"])

	modelsDest := filepath.Join(home, ".pi", "agent", "models.json")
	var models map[string]any
	require.NoError(t, json.Unmarshal(seen[modelsDest].Content, &models))
	providers := models["providers"].(map[string]any)
	gw := providers["hs-gw"].(map[string]any)
	assert.Equal(t, "https://gw", gw["baseUrl"])
	assert.Equal(t, "openai-completions", gw["api"])
	assert.Equal(t, "dummy", gw["apiKey"])
	modelList := gw["models"].([]any)
	first := modelList[0].(map[string]any)
	assert.Equal(t, "claude-sonnet-4-6", first["id"])
	assert.Equal(t, "Sonnet", first["name"])
}

func TestPiRenderMergesExistingSettings(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".pi", "agent")
	require.NoError(t, os.MkdirAll(base, 0o750))

	existing := map[string]any{
		"theme":                "light",
		"defaultThinkingLevel": "high",
	}
	body, _ := json.Marshal(existing)
	require.NoError(t, os.WriteFile(filepath.Join(base, "settings.json"), body, 0o600))

	ad := New(WithHome(home))
	b := &canonical.Bundle{Profile: canonical.Profile{Gateway: canonical.Gateway{URL: "https://gw", DefaultModel: "sonnet"}}}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(seen[filepath.Join(base, "settings.json")].Content, &parsed))
	assert.Equal(t, "light", parsed["theme"])
	assert.Equal(t, "high", parsed["defaultThinkingLevel"])
	assert.Equal(t, "hs-gw", parsed["defaultProvider"])
	assert.Equal(t, "sonnet", parsed["defaultModel"])
}

func TestPiRenderWithoutGatewaySkipsProviderFiles(t *testing.T) {
	home := t.TempDir()
	ad := New(WithHome(home))
	b := &canonical.Bundle{Root: "/canon"}
	fs, err := ad.Render(b)
	require.NoError(t, err)

	seen := map[string]adapter.File{}
	fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

	assert.Contains(t, seen, filepath.Join(home, ".pi", "agent", "skills"))
	assert.NotContains(t, seen, filepath.Join(home, ".pi", "agent", "settings.json"))
	assert.NotContains(t, seen, filepath.Join(home, ".pi", "agent", "models.json"))
}

func TestPiImport(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, ".pi", "agent")
	require.NoError(t, os.MkdirAll(base, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(base, "AGENTS.md"), []byte("# global"), 0o600))

	ad := New(WithHome(home))
	res, err := ad.Import(home)
	require.NoError(t, err)
	assert.Equal(t, "# global", res.Instructions)
}
