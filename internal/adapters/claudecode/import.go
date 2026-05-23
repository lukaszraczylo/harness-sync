package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func importFrom(home string) (*adapter.ImportResult, error) {
	base := filepath.Join(home, ".claude")
	res := &adapter.ImportResult{}

	skills, err := importMarkdownTree(filepath.Join(base, "skills"), "SKILL.md")
	if err != nil {
		return nil, err
	}
	for _, s := range skills {
		res.Skills = append(res.Skills, canonical.Skill{
			Name:        s.name,
			Description: s.description,
			Body:        s.body,
			Path:        s.path,
		})
	}

	agents, err := importMarkdownTree(filepath.Join(base, "agents"), "")
	if err != nil {
		return nil, err
	}
	for _, a := range agents {
		res.Agents = append(res.Agents, canonical.Agent{
			Name:        a.name,
			Description: a.description,
			Body:        a.body,
			Path:        a.path,
		})
	}

	body, err := readIfExists(filepath.Join(base, "CLAUDE.md"))
	if err != nil {
		return nil, err
	}
	res.Instructions = body

	servers, err := importMCPFromSettings(filepath.Join(base, "settings.json"))
	if err != nil {
		return nil, err
	}
	res.MCP = servers
	return res, nil
}

type mdDoc struct {
	name        string
	description string
	body        string
	path        string
}

func importMarkdownTree(dir, requiredFilename string) ([]mdDoc, error) {
	var docs []mdDoc
	if !dirExists(dir) {
		return docs, nil
	}
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if requiredFilename != "" && filepath.Base(p) != requiredFilename {
			return nil
		}
		if requiredFilename == "" && !strings.HasSuffix(p, ".md") {
			return nil
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		n, desc := parseFrontmatter(body)
		if n == "" {
			n = strings.TrimSuffix(filepath.Base(p), ".md")
		}
		rel, _ := filepath.Rel(dir, p)
		docs = append(docs, mdDoc{
			name:        n,
			description: desc,
			body:        string(body),
			path:        rel,
		})
		return nil
	})
	return docs, err
}

type settingsServer struct { //nolint:govet // fieldalignment not achievable: map+slice+3 strings = 80 bytes regardless of order
	Env       map[string]string `json:"env"`
	Args      []string          `json:"args"`
	Command   string            `json:"command"`
	URL       string            `json:"url"`
	Transport string            `json:"transport"`
}

type settingsFile struct {
	MCPServers map[string]settingsServer `json:"mcpServers"`
}

func importMCPFromSettings(path string) ([]canonical.MCPServer, error) {
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var d settingsFile
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, err
	}
	out := make([]canonical.MCPServer, 0, len(d.MCPServers))
	for nm, v := range d.MCPServers {
		out = append(out, canonical.MCPServer{
			Name:      nm,
			Command:   v.Command,
			Args:      v.Args,
			URL:       v.URL,
			Transport: v.Transport,
			Env:       v.Env,
		})
	}
	return out, nil
}

func parseFrontmatter(b []byte) (n, description string) {
	s := string(b)
	if !strings.HasPrefix(s, "---\n") {
		return "", ""
	}
	end := strings.Index(s[4:], "\n---")
	if end == -1 {
		return "", ""
	}
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	_ = yaml.Unmarshal([]byte(s[4:4+end]), &meta)
	return meta.Name, meta.Description
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

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
