package crush

import (
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	cfgPath := filepath.Join(home, ".config", "crush", "crush.json")
	servers, err := common.ImportMCPFromJSONFile(cfgPath, "mcpServers")
	if err != nil {
		return nil, err
	}
	return &adapter.ImportResult{MCP: servers}, nil
}
