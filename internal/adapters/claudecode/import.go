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

	rules, err := common.ImportMarkdownTree(filepath.Join(base, "rules"), "")
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		res.Rules = append(res.Rules, canonical.Rule{
			Name: r.Name, Description: r.Description, Body: r.Body, Path: r.Path,
		})
	}

	body, err := common.ReadIfExists(filepath.Join(base, "CLAUDE.md"))
	if err != nil {
		return nil, err
	}
	res.Instructions = body

	// Claude Code stores MCP servers across several files. Read all known
	// locations and dedupe by name; ~/.claude.json is the live source so
	// its entries win on conflict.
	mcpSources := []string{
		filepath.Join(home, ".claude.json"),
		filepath.Join(base, "mcp_servers.json"),
		filepath.Join(base, "settings.json"),
	}
	seen := map[string]bool{}
	for _, p := range mcpSources {
		servers, err := common.ImportMCPFromJSONFile(p, "mcpServers")
		if err != nil {
			return nil, err
		}
		for _, s := range servers {
			if seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			res.MCP = append(res.MCP, s)
		}
	}
	return res, nil
}
