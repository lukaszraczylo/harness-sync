package claudecode

import (
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	base := filepath.Join(home, ".claude")
	res := &adapter.ImportResult{}

	skills, err := common.ImportMarkdownTree(filepath.Join(base, "skills"), "SKILL.md")
	if err != nil {
		return nil, err
	}
	for _, s := range skills {
		res.Skills = append(res.Skills, canonical.Skill{
			Name: s.Name, Description: s.Description, Body: s.Body, Path: s.Path,
		})
	}

	agents, err := common.ImportMarkdownTree(filepath.Join(base, "agents"), "")
	if err != nil {
		return nil, err
	}
	for _, a := range agents {
		res.Agents = append(res.Agents, canonical.Agent{
			Name: a.Name, Description: a.Description, Body: a.Body, Path: a.Path,
		})
	}

	body, err := common.ReadIfExists(filepath.Join(base, "CLAUDE.md"))
	if err != nil {
		return nil, err
	}
	res.Instructions = body

	servers, err := common.ImportMCPFromJSONFile(filepath.Join(base, "settings.json"), "mcpServers")
	if err != nil {
		return nil, err
	}
	res.MCP = servers
	return res, nil
}
