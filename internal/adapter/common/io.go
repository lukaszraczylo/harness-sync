package common

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

// ReadIfExists returns the file body or empty string when missing.
func ReadIfExists(path string) (string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DirExists reports whether path is an existing directory.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

var (
	blockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	lineComment  = regexp.MustCompile(`(?m)//[^\n]*`)
)

// StripJSONComments is a naive JSONC -> JSON stripper. It does not understand
// strings containing // or /* and is therefore only safe for hand-edited
// config files that don't put comment markers inside string literals.
func StripJSONComments(s string) string {
	s = blockComment.ReplaceAllString(s, "")
	s = lineComment.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// ImportMCPFromJSONFile reads a JSON file and extracts an MCP server map
// nested under the given top-level key (e.g. "mcpServers", "mcp").
// Returns nil with no error when the file is missing.
func ImportMCPFromJSONFile(path, key string) ([]canonical.MCPServer, error) {
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ParseMCPFromJSON(body, key)
}

// ParseMCPFromJSON extracts MCP servers from a JSON body. The body may be
// raw JSON or JSONC (comments are stripped first).
func ParseMCPFromJSON(body []byte, key string) ([]canonical.MCPServer, error) {
	clean := StripJSONComments(string(body))
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(clean), &raw); err != nil {
		return nil, nil // tolerate malformed config
	}
	nested, ok := raw[key]
	if !ok {
		return nil, nil
	}
	var entries map[string]struct { //nolint:govet // fieldalignment: map+slice+3 strings irreducible
		Env       map[string]string `json:"env"`
		Args      []string          `json:"args"`
		Command   string            `json:"command"`
		URL       string            `json:"url"`
		Transport string            `json:"transport"`
	}
	if err := json.Unmarshal(nested, &entries); err != nil {
		return nil, nil
	}
	out := make([]canonical.MCPServer, 0, len(entries))
	for n, v := range entries {
		out = append(out, canonical.MCPServer{
			Name: n, Command: v.Command, Args: v.Args, URL: v.URL,
			Transport: v.Transport, Env: v.Env,
		})
	}
	return out, nil
}
