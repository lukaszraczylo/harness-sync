package canonical

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"

	"github.com/lukaszraczylo/harness-sync/internal/fsx"
)

// Load loads a canonical Bundle from the given root directory using the OS
// filesystem. It is a thin wrapper around LoadFS.
func Load(root string) (*Bundle, error) {
	return LoadFS(fsx.OS(), root)
}

// LoadFS loads a canonical Bundle from root using the provided filesystem.
// Use fsx.Mem() in tests to avoid touching real disk.
func LoadFS(fs fsx.FS, root string) (*Bundle, error) {
	info, err := fs.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("canonical root %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("canonical root %s is not a directory", root)
	}

	b := &Bundle{Root: root}

	err = loadYAMLFS(fs, filepath.Join(root, "config.yaml"), &b.Config)
	if err != nil {
		return nil, err
	}
	if b.Config.ActiveProfile == "" {
		return nil, fmt.Errorf("config.yaml: active_profile is required")
	}

	profPath := filepath.Join(root, "profiles", b.Config.ActiveProfile+".yaml")
	err = loadYAMLFS(fs, profPath, &b.Profile)
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", b.Config.ActiveProfile, err)
	}
	if b.Profile.Name == "" {
		return nil, fmt.Errorf("profile %q (%s): name is required (expected key 'name' with a string value)", b.Config.ActiveProfile, profPath)
	}

	mcpPath := filepath.Join(root, "mcp.yaml")
	if fileExistsFS(fs, mcpPath) {
		err = loadYAMLFS(fs, mcpPath, &b.MCP)
		if err != nil {
			return nil, err
		}
	}

	// Source directories honour Config.Paths overrides (relative to root, or
	// absolute), falling back to the conventional layout.
	skillsDir := resolvePath(root, b.Config.Paths.Skills, "skills")
	agentsDir := resolvePath(root, b.Config.Paths.Agents, "agents")
	instructionsDir := resolvePath(root, b.Config.Paths.Instructions, "instructions")

	skills, err := loadMarkdownDirFS(fs, skillsDir, "SKILL.md")
	if err != nil {
		return nil, err
	}
	for _, m := range skills {
		b.Skills = append(b.Skills, Skill(m))
	}

	agents, err := loadMarkdownDirFS(fs, agentsDir, "")
	if err != nil {
		return nil, err
	}
	for _, m := range agents {
		b.Agents = append(b.Agents, Agent(m))
	}

	b.Instructions.PerHarness = map[string]string{}
	body, err := readFileIfExistsFS(fs, filepath.Join(instructionsDir, "global.md"))
	if err != nil {
		return nil, err
	}
	b.Instructions.Global = body

	perHarnessDir := filepath.Join(instructionsDir, "per-harness")
	if dirExistsFS(fs, perHarnessDir) {
		dir, err := fs.Open(perHarnessDir)
		if err != nil {
			return nil, err
		}
		entries, err := dir.Readdir(-1)
		if closeErr := dir.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			entryBody, err := afero.ReadFile(fs, filepath.Join(perHarnessDir, e.Name()))
			if err != nil {
				return nil, err
			}
			harness := strings.TrimSuffix(e.Name(), ".md")
			b.Instructions.PerHarness[harness] = string(entryBody)
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

// loadMarkdownDirFS walks dir on fs and returns one markdownDoc per matching file.
func loadMarkdownDirFS(fs fsx.FS, dir, requiredFilename string) ([]markdownDoc, error) {
	var docs []markdownDoc
	if !dirExistsFS(fs, dir) {
		return docs, nil
	}
	err := afero.Walk(fs, dir, func(path string, info os.FileInfo, err error) error {
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
		body, err := afero.ReadFile(fs, path)
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

// resolvePath returns custom (relative to root, or used as-is when absolute)
// when set, else the conventional path under root from defaults.
func resolvePath(root, custom string, defaults ...string) string {
	if custom != "" {
		if filepath.IsAbs(custom) {
			return custom
		}
		return filepath.Join(root, custom)
	}
	return filepath.Join(append([]string{root}, defaults...)...)
}

func parseFrontmatter(b []byte) (name, description string) {
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
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
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return "", ""
	}
	return meta.Name, meta.Description
}

func loadYAMLFS(fs fsx.FS, path string, out any) error {
	b, err := afero.ReadFile(fs, path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	// Strict decode: reject unknown/misspelled keys instead of silently leaving
	// the corresponding field at its zero value (which would render a wrong
	// config for the user's tools). io.EOF means an empty document — fine.
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("parse %s: %w (check YAML syntax and field names)", path, err)
	}
	return nil
}

func readFileIfExistsFS(fs fsx.FS, path string) (string, error) {
	b, err := afero.ReadFile(fs, path)
	if err != nil && os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func fileExistsFS(fs fsx.FS, path string) bool {
	info, err := fs.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExistsFS(fs fsx.FS, path string) bool {
	info, err := fs.Stat(path)
	return err == nil && info.IsDir()
}
