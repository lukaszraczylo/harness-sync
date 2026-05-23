package common

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Doc mirrors canonical.Skill / canonical.Agent without importing them.
type Doc struct {
	Name        string
	Description string
	Body        string
	Path        string
}

// ParseFrontmatter extracts the name and description fields from a markdown
// file's YAML frontmatter block. Returns ("", "") when there is no frontmatter.
func ParseFrontmatter(b []byte) (name, description string) {
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

// ImportMarkdownTree walks dir and returns one Doc per markdown file. When
// requiredFilename is non-empty (e.g. "SKILL.md"), only files with that exact
// basename are included; otherwise any *.md.
func ImportMarkdownTree(dir, requiredFilename string) ([]Doc, error) {
	var docs []Doc
	if !DirExists(dir) {
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
		name, desc := ParseFrontmatter(body)
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(p), ".md")
		}
		rel, _ := filepath.Rel(dir, p)
		docs = append(docs, Doc{
			Name:        name,
			Description: desc,
			Body:        string(body),
			Path:        rel,
		})
		return nil
	})
	return docs, err
}
