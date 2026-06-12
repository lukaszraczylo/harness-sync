package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func reg(servers ...canonical.MCPServer) *canonical.MCPRegistry {
	return &canonical.MCPRegistry{Servers: servers}
}

func TestBuildMCPMapStyledClaudeStyle(t *testing.T) {
	r := reg(canonical.MCPServer{Name: "fp", Command: "/bin/fp", Args: []string{"--flag"}})
	out := BuildMCPMapStyled(r, MCPClaudeStyle)
	require.Contains(t, out, "fp")
	e := out["fp"].(map[string]any)
	assert.Equal(t, "/bin/fp", e["command"])
	assert.Equal(t, []string{"--flag"}, e["args"])
	// Claude's actual entry shape uses "type" (defaults to stdio when no URL).
	assert.Equal(t, "stdio", e["type"])
	assert.NotContains(t, e, "transport")
}

func TestBuildMCPMapStyledClaudeStyleURL(t *testing.T) {
	r := reg(canonical.MCPServer{
		Name:    "remote",
		URL:     "https://example.com",
		Headers: map[string]string{"Authorization": "Bearer ${MCP_TOKEN}"},
	})
	out := BuildMCPMapStyled(r, MCPClaudeStyle)
	e := out["remote"].(map[string]any)
	assert.Equal(t, "https://example.com", e["url"])
	assert.Equal(t, "http", e["type"])
	assert.Equal(t, map[string]string{"Authorization": "Bearer ${MCP_TOKEN}"}, e["headers"])
}

func TestBuildMCPMapStyledCrushStdio(t *testing.T) {
	r := reg(canonical.MCPServer{Name: "fp", Command: "/bin/fp", Args: []string{"-x"}})
	out := BuildMCPMapStyled(r, MCPCrushStyle)
	e := out["fp"].(map[string]any)
	assert.Equal(t, "stdio", e["type"])
	assert.Equal(t, "/bin/fp", e["command"])
	assert.Equal(t, []string{"-x"}, e["args"])
}

func TestBuildMCPMapStyledCrushHTTP(t *testing.T) {
	r := reg(canonical.MCPServer{Name: "remote", URL: "https://example.com", Transport: "sse"})
	out := BuildMCPMapStyled(r, MCPCrushStyle)
	e := out["remote"].(map[string]any)
	assert.Equal(t, "sse", e["type"])
	assert.Equal(t, "https://example.com", e["url"])
}

func TestBuildMCPMapStyledCrushHTTPDefaultType(t *testing.T) {
	// No transport set → default to "http"
	r := reg(canonical.MCPServer{Name: "remote", URL: "https://example.com"})
	out := BuildMCPMapStyled(r, MCPCrushStyle)
	e := out["remote"].(map[string]any)
	assert.Equal(t, "http", e["type"])
}

func TestBuildMCPMapStyledOpencodeLocal(t *testing.T) {
	r := reg(canonical.MCPServer{Name: "fp", Command: "/bin/fp", Args: []string{"--flag"}})
	out := BuildMCPMapStyled(r, MCPOpencodeStyle)
	e := out["fp"].(map[string]any)
	assert.Equal(t, "local", e["type"])
	assert.Equal(t, true, e["enabled"])
	cmd := e["command"].([]string)
	assert.Equal(t, []string{"/bin/fp", "--flag"}, cmd)
	assert.NotContains(t, e, "environment")
}

func TestBuildMCPMapStyledOpencodeRemote(t *testing.T) {
	r := reg(canonical.MCPServer{
		Name:    "r",
		URL:     "https://example.com",
		Headers: map[string]string{"X-Tenant": "tenant-a"},
	})
	out := BuildMCPMapStyled(r, MCPOpencodeStyle)
	e := out["r"].(map[string]any)
	assert.Equal(t, "remote", e["type"])
	assert.Equal(t, true, e["enabled"])
	assert.Equal(t, "https://example.com", e["url"])
	assert.Equal(t, map[string]string{"X-Tenant": "tenant-a"}, e["headers"])
}

func TestBuildMCPMapStyledOpencodeEnv(t *testing.T) {
	r := reg(canonical.MCPServer{Name: "fp", Command: "/bin/fp", Env: map[string]string{"K": "V"}})
	out := BuildMCPMapStyled(r, MCPOpencodeStyle)
	e := out["fp"].(map[string]any)
	assert.Equal(t, map[string]string{"K": "V"}, e["environment"])
}

func TestBuildMCPMapStyledZedStdio(t *testing.T) {
	r := reg(canonical.MCPServer{Name: "fp", Command: "/bin/fp", Args: []string{"-x"}})
	out := BuildMCPMapStyled(r, MCPZedStyle)
	e := out["fp"].(map[string]any)
	// Zed 1.3.x serde bug: "enabled" and "source" must NOT be present.
	assert.NotContains(t, e, "enabled")
	assert.NotContains(t, e, "source")
	assert.Equal(t, "/bin/fp", e["command"])
	assert.Equal(t, []string{"-x"}, e["args"])
}

func TestBuildMCPMapStyledZedURL(t *testing.T) {
	r := reg(canonical.MCPServer{Name: "r", URL: "https://example.com"})
	out := BuildMCPMapStyled(r, MCPZedStyle)
	e := out["r"].(map[string]any)
	// Zed 1.3.x serde bug: "enabled" must NOT be present.
	assert.NotContains(t, e, "enabled")
	assert.Equal(t, "https://example.com", e["url"])
	assert.NotContains(t, e, "source")
}

func TestBuildMCPMapStyledNilReg(t *testing.T) {
	out := BuildMCPMapStyled(nil, MCPClaudeStyle)
	assert.NotNil(t, out)
	assert.Len(t, out, 0)
}
