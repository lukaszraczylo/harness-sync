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
// raw JSON or JSONC (comments are stripped first). Tolerates the dialect
// differences across harnesses:
//   - claude / crush: command is a string, transport key is "type"
//   - kilo / opencode: command is a []string (first element is the binary)
//   - some configs: env appears as "env", others as "environment"
//   - some configs: transport appears as "type", others as "transport"
func ParseMCPFromJSON(body []byte, key string) ([]canonical.MCPServer, error) {
	// Try plain JSON first — StripJSONComments eats // inside URL strings.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		clean := StripJSONComments(string(body))
		if err := json.Unmarshal([]byte(clean), &raw); err != nil {
			return nil, nil
		}
	}
	nested, ok := raw[key]
	if !ok {
		return nil, nil
	}
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(nested, &entries); err != nil {
		return nil, nil
	}
	out := make([]canonical.MCPServer, 0, len(entries))
	for n, raw := range entries {
		srv, ok := parseMCPEntry(n, raw)
		if !ok {
			continue
		}
		// Skip extension-managed entries: they have no command and no url
		// (e.g. zed extensions providing their own MCP). Nothing to propagate.
		if srv.Command == "" && srv.URL == "" {
			continue
		}
		out = append(out, srv)
	}
	return out, nil
}

func parseMCPEntry(name string, raw json.RawMessage) (canonical.MCPServer, bool) {
	var e struct { //nolint:govet // fieldalignment: dialect-tolerant entry, irreducible
		Type        string            `json:"type"`
		Transport   string            `json:"transport"`
		Command     json.RawMessage   `json:"command"`
		Cmd         json.RawMessage   `json:"cmd"`
		Args        []string          `json:"args"`
		URL         string            `json:"url"`
		Env         map[string]string `json:"env"`
		Environment map[string]string `json:"environment"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return canonical.MCPServer{}, false
	}

	cmdRaw := e.Command
	if len(cmdRaw) == 0 {
		cmdRaw = e.Cmd
	}
	cmd, args := splitCommand(cmdRaw)
	args = append(args, e.Args...)

	env := e.Env
	if len(env) == 0 {
		env = e.Environment
	}

	transport := e.Type
	if transport == "" {
		transport = e.Transport
	}

	return canonical.MCPServer{
		Name:      name,
		Command:   cmd,
		Args:      args,
		URL:       e.URL,
		Transport: transport,
		Env:       env,
	}, true
}

// splitCommand accepts a JSON value that may be a string ("foo") or a
// string array (["foo","--bar"]). Returns (head, tail).
func splitCommand(raw json.RawMessage) (string, []string) {
	if len(raw) == 0 {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr[0], arr[1:]
	}
	return "", nil
}
