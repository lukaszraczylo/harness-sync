package common

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"

	"github.com/lukaszraczylo/harness-sync/internal/fsx"
)

// Doc mirrors canonical.Skill / canonical.Agent without importing them.
type Doc struct {
	Name        string
	Description string
	Body        string
	Path        string
}

// Managed-block markers delimit the region of an instruction file that
// harness-sync owns. Everything outside the markers is user content and is
// preserved verbatim across applies.
const (
	ManagedBlockBegin = "<!-- BEGIN harness-sync (managed) — do not edit between markers -->"
	ManagedBlockEnd   = "<!-- END harness-sync (managed) -->"
)

// MergeManagedMarkdown returns existing content with the harness-sync managed
// block inserted or replaced, preserving all user content outside the markers.
//   - No existing content: returns just the managed block.
//   - Existing markers present: replaces only the region between them.
//   - Existing content without markers: appends the managed block after the
//     user's content (never overwrites it).
//
// managed is the canonical instruction text harness-sync owns.
func MergeManagedMarkdown(existing, managed string) string {
	block := ManagedBlockBegin + "\n" + strings.TrimRight(managed, "\n") + "\n" + ManagedBlockEnd
	if strings.TrimSpace(existing) == "" {
		return block + "\n"
	}
	bi := strings.Index(existing, ManagedBlockBegin)
	ei := strings.Index(existing, ManagedBlockEnd)
	if bi >= 0 && ei > bi {
		return existing[:bi] + block + existing[ei+len(ManagedBlockEnd):]
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + block + "\n"
}

// ParseFrontmatter extracts the name and description fields from a markdown
// file's YAML frontmatter block. Returns ("", "") when there is no frontmatter.
// CRLF line endings are normalised so Windows-authored files parse correctly.
func ParseFrontmatter(b []byte) (name, description string) {
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
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
	if err := yaml.Unmarshal([]byte(s[4:4+end]), &meta); err != nil {
		return "", ""
	}
	return meta.Name, meta.Description
}

// ImportMarkdownTreeFS walks dir on fs and returns one Doc per matching file.
// When requiredFilename is non-empty (e.g. "SKILL.md"), only files with that
// exact basename are included; otherwise any *.md.
func ImportMarkdownTreeFS(fs fsx.FS, dir, requiredFilename string) ([]Doc, error) {
	var docs []Doc
	if !DirExistsFS(fs, dir) {
		return docs, nil
	}
	err := afero.Walk(fs, dir, func(p string, info os.FileInfo, err error) error {
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
		body, err := afero.ReadFile(fs, p)
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

// ImportMarkdownTree walks dir and returns one Doc per markdown file. When
// requiredFilename is non-empty (e.g. "SKILL.md"), only files with that exact
// basename are included; otherwise any *.md.
func ImportMarkdownTree(dir, requiredFilename string) ([]Doc, error) {
	return ImportMarkdownTreeFS(fsx.OS(), dir, requiredFilename)
}
