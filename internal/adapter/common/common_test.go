package common

import (
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
