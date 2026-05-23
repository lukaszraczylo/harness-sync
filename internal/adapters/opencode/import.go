package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	base := filepath.Join(home, ".config", "opencode")
	res := &adapter.ImportResult{}

	cfgBody, err := readIfExists(filepath.Join(base, "opencode.jsonc"))
	if err != nil {
		return nil, err
	}
	if cfgBody != "" {
		clean := stripJSONComments(cfgBody)
		var doc struct {
			MCPServers map[string]struct { //nolint:govet // fieldalignment: map+slice+3 strings irreducible
				Env       map[string]string `json:"env"`
				Args      []string          `json:"args"`
				Command   string            `json:"command"`
				URL       string            `json:"url"`
				Transport string            `json:"transport"`
			} `json:"mcpServers"`
		}
		_ = json.Unmarshal([]byte(clean), &doc)
		for nm, v := range doc.MCPServers {
			res.MCP = append(res.MCP, canonical.MCPServer{
				Name:      nm,
				Command:   v.Command,
				Args:      v.Args,
				URL:       v.URL,
				Transport: v.Transport,
				Env:       v.Env,
			})
		}
	}

	body, err := readIfExists(filepath.Join(base, "AGENTS.md"))
	if err != nil {
		return nil, err
	}
	res.Instructions = body

	return res, nil
}

var (
	blockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	lineComment  = regexp.MustCompile(`(?m)//[^\n]*`)
)

func stripJSONComments(s string) string {
	s = blockComment.ReplaceAllString(s, "")
	s = lineComment.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func readIfExists(path string) (string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}
