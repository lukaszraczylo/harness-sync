package zed

import (
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	cfgPath := filepath.Join(home, ".config", "zed", "settings.json")
	// context_servers entries use source:"custom" for stdio; the MCP command
	// field maps directly to canonical MCPServer.Command.
	servers, err := common.ImportMCPFromJSONFile(cfgPath, "context_servers")
	if err != nil {
		return nil, err
	}
	return &adapter.ImportResult{MCP: servers}, nil
}
