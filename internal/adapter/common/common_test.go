package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestBuildMCPMap(t *testing.T) {
	reg := &canonical.MCPRegistry{Servers: []canonical.MCPServer{
		{Name: "filepuff", Command: "/bin/x", Transport: "stdio"},
	}}
	out := BuildMCPMap(reg)
	assert.Contains(t, out, "filepuff")
	entry := out["filepuff"].(map[string]any)
	assert.Equal(t, "/bin/x", entry["command"])
}

func TestBuildMCPMapNil(t *testing.T) {
	out := BuildMCPMap(nil)
	assert.NotNil(t, out)
	assert.Len(t, out, 0)
}

func TestStripJSONComments(t *testing.T) {
	s := "// hi\n{\"a\":1, /* x */ \"b\":2}"
	assert.Equal(t, `{"a":1,  "b":2}`, StripJSONComments(s))
}

func TestParseFrontmatter(t *testing.T) {
	body := []byte("---\nname: foo\ndescription: bar\n---\nbody")
	name, desc := ParseFrontmatter(body)
	assert.Equal(t, "foo", name)
	assert.Equal(t, "bar", desc)
}

func TestParseFrontmatterMissing(t *testing.T) {
	name, desc := ParseFrontmatter([]byte("no frontmatter"))
	assert.Empty(t, name)
	assert.Empty(t, desc)
}

func TestImportMarkdownTreeMissingDir(t *testing.T) {
	docs, err := ImportMarkdownTree("/nonexistent/path", "")
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestParseMCPFromJSON(t *testing.T) {
	body := []byte(`{"mcpServers": {"filepuff": {"command": "/bin/fp", "args": ["--flag"]}}}`)
	servers, err := ParseMCPFromJSON(body, "mcpServers")
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "filepuff", servers[0].Name)
	assert.Equal(t, "/bin/fp", servers[0].Command)
}

func TestParseMCPFromJSONMissingKey(t *testing.T) {
	body := []byte(`{"other": {}}`)
	servers, err := ParseMCPFromJSON(body, "mcpServers")
	require.NoError(t, err)
	assert.Nil(t, servers)
}

func TestProvidersAsMapHonorsPerModelLimits(t *testing.T) {
	p := &canonical.Profile{
		Gateway: canonical.Gateway{URL: "https://gw.example.com/v1", Token: "t"},
		Models: []canonical.Model{
			{ID: "prov/big", Context: 262144, Output: 65536},
			{ID: "prov/default"},
		},
	}
	models := ProvidersAsMap(p)[GatewayProviderKey(p.Gateway.URL)].(map[string]any)["models"].(map[string]any)

	big := models["prov/big"].(map[string]any)["limit"].(map[string]any)
	assert.Equal(t, 262144, big["context"], "explicit context override should win")
	assert.Equal(t, 65536, big["output"], "explicit output override should win")

	def := models["prov/default"].(map[string]any)["limit"].(map[string]any)
	assert.Equal(t, 200000, def["context"], "unset model falls back to default context")
	assert.Equal(t, 8192, def["output"], "unset model falls back to default output")
}

func TestProvidersAsCrushMapHonorsPerModelLimits(t *testing.T) {
	p := &canonical.Profile{
		Gateway: canonical.Gateway{URL: "https://gw.example.com/v1", Token: "t"},
		Models: []canonical.Model{
			{ID: "prov/big", Context: 262144, Output: 65536},
			{ID: "prov/default"},
		},
	}
	models := ProvidersAsCrushMap(p)[GatewayProviderKey(p.Gateway.URL)].(map[string]any)["models"].([]map[string]any)

	assert.Equal(t, 262144, models[0]["context_window"], "explicit context override should win")
	assert.Equal(t, 65536, models[0]["default_max_tokens"], "explicit output override should win")
	assert.Equal(t, 200000, models[1]["context_window"], "unset model falls back to default context")
	assert.Equal(t, 8192, models[1]["default_max_tokens"], "unset model falls back to default output")
}

func TestGooseCustomProviderFileHonorsPerModelContext(t *testing.T) {
	p := &canonical.Profile{
		Gateway: canonical.Gateway{URL: "https://gw.example.com/v1", Token: "t"},
		Models:  []canonical.Model{{ID: "big", Context: 262144}, {ID: "default"}},
	}
	body, _ := GooseCustomProviderFile(p)
	var entry map[string]any
	require.NoError(t, json.Unmarshal(body, &entry))
	models := entry["models"].([]any)

	assert.Equal(t, float64(262144), models[0].(map[string]any)["context_limit"], "explicit context override should win")
	assert.Equal(t, float64(200000), models[1].(map[string]any)["context_limit"], "unset model falls back to default context")
}
