package opencode

import (
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	base := filepath.Join(home, ".config", "opencode")
	res := &adapter.ImportResult{}

	agents, err := common.ImportMarkdownTree(filepath.Join(base, "agents"), "")
	if err != nil {
		return nil, err
	}
	for _, a := range agents {
		res.Agents = append(res.Agents, canonical.Agent{
			Name: a.Name, Description: a.Description, Body: a.Body, Path: a.Path,
		})
	}

	cfgBody, err := common.ReadIfExists(filepath.Join(base, "opencode.jsonc"))
	if err != nil {
		return nil, err
	}
	if cfgBody != "" {
		mcpServers, mcpErr := common.ParseMCPFromJSON([]byte(cfgBody), "mcp")
		if mcpErr != nil {
			return nil, mcpErr
		}
		res.MCP = mcpServers
	}

	instructions, err := common.ReadIfExists(filepath.Join(base, "AGENTS.md"))
	if err != nil {
		return nil, err
	}
	res.Instructions = instructions
	return res, nil
}
