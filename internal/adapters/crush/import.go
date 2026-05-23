package crush

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

type crushMCPEntry struct { //nolint:govet // fieldalignment: map+slice+3 strings irreducible
	Env       map[string]string `json:"env"`
	Args      []string          `json:"args"`
	Command   string            `json:"command"`
	URL       string            `json:"url"`
	Transport string            `json:"transport"`
}

type crushConfig struct {
	MCPServers map[string]crushMCPEntry `json:"mcpServers"`
}

func importFrom(home string) (*adapter.ImportResult, error) {
	cfgPath := filepath.Join(home, ".config", "crush", "crush.json")
	body, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		return &adapter.ImportResult{}, nil
	}
	if err != nil {
		return nil, err
	}

	var doc crushConfig
	_ = json.Unmarshal(body, &doc)

	res := &adapter.ImportResult{}
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
	return res, nil
}
