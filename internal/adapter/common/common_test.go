package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestBuildProvidersGatewayFirst(t *testing.T) {
	p := &canonical.Profile{
		Gateway: canonical.Gateway{URL: "https://gw", Token: "t", DefaultModel: "x"},
		Models:  []canonical.Model{{ID: "m1", Alias: "a1"}},
		Upstreams: []canonical.Upstream{
			{Name: "anthropic", APIKey: "k", BaseURL: "https://api.anthropic.com"},
		},
	}
	out := BuildProviders(p)
	require.Len(t, out, 2)
	assert.Equal(t, "harness-sync-gateway", out[0]["id"])
	assert.Equal(t, "anthropic", out[1]["id"])
	assert.NotNil(t, out[0]["models"])
}

func TestBuildProvidersNoGateway(t *testing.T) {
	p := &canonical.Profile{
		Upstreams: []canonical.Upstream{
			{Name: "anthropic", APIKey: "k"},
		},
	}
	out := BuildProviders(p)
	require.Len(t, out, 1)
	assert.Equal(t, "anthropic", out[0]["id"])
}

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
