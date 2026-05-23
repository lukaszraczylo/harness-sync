package canonical

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(root string) (*Bundle, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("canonical root %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("canonical root %s is not a directory", root)
	}

	b := &Bundle{Root: root}

	err = loadYAML(filepath.Join(root, "config.yaml"), &b.Config)
	if err != nil {
		return nil, err
	}
	if b.Config.ActiveProfile == "" {
		return nil, fmt.Errorf("config.yaml: active_profile is required")
	}

	profPath := filepath.Join(root, "profiles", b.Config.ActiveProfile+".yaml")
	err = loadYAML(profPath, &b.Profile)
	if err != nil {
		return nil, err
	}

	mcpPath := filepath.Join(root, "mcp.yaml")
	if fileExists(mcpPath) {
		err = loadYAML(mcpPath, &b.MCP)
		if err != nil {
			return nil, err
		}
	}

	skills, err := loadMarkdownDir(filepath.Join(root, "skills"), "SKILL.md")
	if err != nil {
		return nil, err
	}
	for _, m := range skills {
		b.Skills = append(b.Skills, Skill(m))
	}

	agents, err := loadMarkdownDir(filepath.Join(root, "agents"), "")
	if err != nil {
		return nil, err
	}
	for _, m := range agents {
		b.Agents = append(b.Agents, Agent(m))
	}

	b.Instructions.PerHarness = map[string]string{}
	body, err := readFileIfExists(filepath.Join(root, "instructions", "global.md"))
	if err != nil {
		return nil, err
	}
	b.Instructions.Global = body

	perHarnessDir := filepath.Join(root, "instructions", "per-harness")
	if entries, err := os.ReadDir(perHarnessDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			body, err := os.ReadFile(filepath.Join(perHarnessDir, e.Name()))
			if err != nil {
				return nil, err
			}
			harness := strings.TrimSuffix(e.Name(), ".md")
			b.Instructions.PerHarness[harness] = string(body)
		}
	}

	return b, nil
}

type markdownDoc struct {
	Name        string
	Description string
	Body        string
	Path        string
}

func loadMarkdownDir(dir, requiredFilename string) ([]markdownDoc, error) {
	var docs []markdownDoc
	if !dirExists(dir) {
		return docs, nil
	}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if requiredFilename != "" && filepath.Base(path) != requiredFilename {
			return nil
		}
		if requiredFilename == "" && !strings.HasSuffix(path, ".md") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		name, desc := parseFrontmatter(body)
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(path), ".md")
		}
		docs = append(docs, markdownDoc{
			Name:        name,
			Description: desc,
			Body:        string(body),
			Path:        rel,
		})
		return nil
	})
	return docs, err
}

func parseFrontmatter(b []byte) (name, description string) {
	s := string(b)
	if !strings.HasPrefix(s, "---\n") {
		return "", ""
	}
	end := strings.Index(s[4:], "\n---")
	if end == -1 {
		return "", ""
	}
	fm := s[4 : 4+end]
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	_ = yaml.Unmarshal([]byte(fm), &meta)
	return meta.Name, meta.Description
}

func loadYAML(path string, out interface{}) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func readFileIfExists(path string) (string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
