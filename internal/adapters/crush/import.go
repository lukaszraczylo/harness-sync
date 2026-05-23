package crush

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	cfgPath := filepath.Join(home, ".config", "crush", "crush.json")
	body, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		return &adapter.ImportResult{}, nil
	}
	if err != nil {
		return nil, err
	}

	var doc struct {
		MCPServers map[string]struct {
			Command   string            `json:"command"`
			Args      []string          `json:"args"`
			URL       string            `json:"url"`
			Transport string            `json:"transport"`
			Env       map[string]string `json:"env"`
		} `json:"mcpServers"`
	}
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
