package kilo

import (
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	cfgPath := filepath.Join(home, ".config", "kilo", "kilo.json")
	servers, err := common.ImportMCPFromJSONFile(cfgPath, "mcp")
	if err != nil {
		return nil, err
	}
	return &adapter.ImportResult{MCP: servers}, nil
}
