# harness-sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Single Go binary that maintains `~/.config/harness-sync/` as a canonical, git-versioned source of truth and propagates skills, agents, MCP, LLM endpoints (via profiles), and global instructions into multiple LLM harnesses with three-way-merge conflict resolution.

**Architecture:** Cobra CLI; canonical YAML+markdown in a git repo; plugin-style adapters per harness (Go interface + static registry); hybrid sync (symlinks for skills/agents, rendered files for LLM/MCP/instructions); conflict resolution via `git merge-file`.

**Tech Stack:** Go 1.22+, cobra (CLI), huh (interactive prompts), goccy/go-yaml (YAML preserving comments), pelletier/go-toml/v2, encoding/json, stretchr/testify (tests).

**Spec:** `docs/superpowers/specs/2026-05-23-harness-sync-design.md`

---

## File Structure

```
harness-sync/
├── cmd/harness-sync/main.go              # cobra entrypoint
├── internal/
│   ├── canonical/                        # canonical struct types + loaders
│   │   ├── types.go
│   │   ├── load.go
│   │   └── load_test.go
│   ├── secrets/                          # ${ENV_VAR} substitution
│   │   ├── envsub.go
│   │   └── envsub_test.go
│   ├── render/                           # format-specific marshallers
│   │   ├── json.go
│   │   ├── yaml.go
│   │   ├── toml.go
│   │   └── render_test.go
│   ├── gitx/                             # thin wrapper over `git` CLI
│   │   ├── repo.go
│   │   └── repo_test.go
│   ├── merge/                            # 3-way merge using git merge-file
│   │   ├── merge.go
│   │   └── merge_test.go
│   ├── adapter/                          # Adapter interface, registry, base helpers
│   │   ├── adapter.go
│   │   ├── registry.go
│   │   ├── fileset.go
│   │   └── adapter_test.go
│   ├── adapters/
│   │   ├── claudecode/
│   │   ├── crush/
│   │   ├── kilo/
│   │   ├── opencode/
│   │   ├── goose/
│   │   ├── cagent/
│   │   └── zed/
│   ├── ui/                               # interactive prompts
│   │   ├── prompts.go
│   │   └── prompts_test.go
│   ├── apply/                            # apply pipeline (orchestration)
│   │   ├── apply.go
│   │   └── apply_test.go
│   └── cli/                              # cobra subcommands
│       ├── root.go
│       ├── init.go
│       ├── detect.go
│       ├── diff.go
│       ├── apply.go
│       ├── rollback.go
│       ├── profile.go
│       └── adapter.go
├── docs/superpowers/specs/2026-05-23-harness-sync-design.md
├── docs/superpowers/plans/2026-05-23-harness-sync.md
├── go.mod
├── Makefile
└── README.md
```

Each file under `internal/adapters/<name>/` follows the pattern:

```
internal/adapters/<name>/
├── <name>.go                             # Adapter impl (Detect, Targets, Render, Import)
├── <name>_test.go                        # golden-file tests
└── testdata/
    ├── canonical_minimal/                # canonical input fixtures
    └── expected/                         # golden outputs
```

---

## Phase 1: Foundations

### Task 1: Bootstrap module

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `README.md`
- Create: `cmd/harness-sync/main.go`

- [ ] **Step 1: Initialise go module**

```bash
go mod init github.com/lukaszraczylo/harness-sync
go get github.com/spf13/cobra@v1.8.1
go get github.com/charmbracelet/huh@v0.5.3
go get github.com/goccy/go-yaml@v1.12.0
go get github.com/pelletier/go-toml/v2@v2.2.3
go get github.com/stretchr/testify@v1.9.0
```

- [ ] **Step 2: Write entrypoint stub**

`cmd/harness-sync/main.go`:

```go
package main

import (
    "fmt"
    "os"

    "github.com/lukaszraczylo/harness-sync/internal/cli"
)

var version = "dev"

func main() {
    if err := cli.NewRoot(version).Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

- [ ] **Step 3: Write minimal cli.NewRoot to keep build green**

`internal/cli/root.go`:

```go
package cli

import "github.com/spf13/cobra"

func NewRoot(version string) *cobra.Command {
    return &cobra.Command{
        Use:     "harness-sync",
        Short:   "Sync skills, agents, MCP, and LLM endpoints across LLM harnesses",
        Version: version,
        SilenceUsage: true,
    }
}
```

- [ ] **Step 4: Write Makefile**

```makefile
.PHONY: build test lint clean install

BIN := harness-sync
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BIN) ./cmd/harness-sync

test:
	go test ./...

lint:
	go vet ./...

install: build
	install -m 0755 $(BIN) $$HOME/.local/bin/$(BIN)

clean:
	rm -f $(BIN)
```

- [ ] **Step 5: Write README skeleton**

`README.md`:

```markdown
# harness-sync

Sync skills, agents, MCP, and LLM endpoints across multiple LLM harnesses
(claude-code, crush, kilo, opencode, goose, cagent, zed).

Canonical config lives in `~/.config/harness-sync/` (git-tracked). Run
`harness-sync apply` to propagate.

See `docs/superpowers/specs/2026-05-23-harness-sync-design.md` for design.

## Build

    make build && make install
```

- [ ] **Step 6: Verify build + run**

Run: `make build && ./harness-sync --version`
Expected: prints `harness-sync version dev` (or similar)

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum cmd/harness-sync internal/cli/root.go Makefile README.md
git commit -m "feat: bootstrap module and cobra root command"
```

---

### Task 2: Canonical types

**Files:**
- Create: `internal/canonical/types.go`
- Create: `internal/canonical/types_test.go`

- [ ] **Step 1: Write the failing test**

`internal/canonical/types_test.go`:

```go
package canonical

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestProfileActiveModel(t *testing.T) {
    p := &Profile{
        Name: "home",
        Gateway: Gateway{
            URL:          "https://gw",
            Token:        "dummy",
            DefaultModel: "sonnet",
        },
        Models: []Model{
            {ID: "claude-sonnet-4-6", Alias: "sonnet"},
            {ID: "claude-opus-4-7", Alias: "opus"},
        },
    }
    m, ok := p.LookupModel("sonnet")
    assert.True(t, ok)
    assert.Equal(t, "claude-sonnet-4-6", m.ID)

    _, ok = p.LookupModel("missing")
    assert.False(t, ok)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/canonical/...`
Expected: FAIL with `undefined: Profile`

- [ ] **Step 3: Implement types**

`internal/canonical/types.go`:

```go
package canonical

type Config struct {
    EnabledHarnesses []string `yaml:"enabled_harnesses"`
    ActiveProfile   string    `yaml:"active_profile"`
    Paths           Paths     `yaml:"paths,omitempty"`
}

type Paths struct {
    Skills       string `yaml:"skills,omitempty"`       // override default ./skills
    Agents       string `yaml:"agents,omitempty"`
    Instructions string `yaml:"instructions,omitempty"`
}

type Profile struct {
    Name      string     `yaml:"name"`
    Gateway   Gateway    `yaml:"gateway"`
    Upstreams []Upstream `yaml:"upstreams,omitempty"`
    Models    []Model    `yaml:"models"`
}

type Gateway struct {
    URL          string `yaml:"url"`
    Token        string `yaml:"token"`
    DefaultModel string `yaml:"default_model"`
}

type Upstream struct {
    Name    string `yaml:"name"`
    APIKey  string `yaml:"api_key,omitempty"`
    BaseURL string `yaml:"base_url,omitempty"`
}

type Model struct {
    ID    string `yaml:"id"`
    Alias string `yaml:"alias,omitempty"`
}

type MCPRegistry struct {
    Servers []MCPServer `yaml:"servers"`
}

type MCPServer struct {
    Name      string            `yaml:"name"`
    Command   string            `yaml:"command,omitempty"`
    Args      []string          `yaml:"args,omitempty"`
    URL       string            `yaml:"url,omitempty"`
    Transport string            `yaml:"transport,omitempty"` // stdio | http | sse
    Env       map[string]string `yaml:"env,omitempty"`
}

type Skill struct {
    Name        string
    Description string
    Body        string // raw markdown including frontmatter
    Path        string // canonical relative path, e.g. "skills/foo/SKILL.md"
}

type Agent struct {
    Name        string
    Description string
    Body        string
    Path        string
}

type Instructions struct {
    Global      string            // markdown body
    PerHarness  map[string]string // harness name -> body
}

type Bundle struct {
    Config       Config
    Profile      Profile
    MCP          MCPRegistry
    Skills       []Skill
    Agents       []Agent
    Instructions Instructions
    Root         string // absolute path to canonical root
}

func (p *Profile) LookupModel(alias string) (Model, bool) {
    for _, m := range p.Models {
        if m.Alias == alias || m.ID == alias {
            return m, true
        }
    }
    return Model{}, false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/canonical/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/canonical
git commit -m "feat(canonical): introduce core domain types"
```

---

### Task 3: Canonical loader

**Files:**
- Create: `internal/canonical/load.go`
- Create: `internal/canonical/load_test.go`
- Create: `internal/canonical/testdata/sample/...`

- [ ] **Step 1: Create test fixture tree**

```bash
mkdir -p internal/canonical/testdata/sample/profiles
mkdir -p internal/canonical/testdata/sample/skills/hello
mkdir -p internal/canonical/testdata/sample/agents
mkdir -p internal/canonical/testdata/sample/instructions
```

Write `internal/canonical/testdata/sample/config.yaml`:

```yaml
enabled_harnesses: [claude-code, crush]
active_profile: home
```

Write `internal/canonical/testdata/sample/profiles/home.yaml`:

```yaml
name: home
gateway:
  url: https://gw.lan
  token: dummy
  default_model: sonnet
models:
  - id: claude-sonnet-4-6
    alias: sonnet
```

Write `internal/canonical/testdata/sample/mcp.yaml`:

```yaml
servers:
  - name: filepuff
    command: /usr/local/bin/filepuff
    transport: stdio
```

Write `internal/canonical/testdata/sample/skills/hello/SKILL.md`:

```markdown
---
name: hello
description: example skill
---
hello body
```

Write `internal/canonical/testdata/sample/agents/sample.md`:

```markdown
---
name: sample
description: sample agent
---
sample body
```

Write `internal/canonical/testdata/sample/instructions/global.md`:

```markdown
# Global instructions

Be helpful.
```

- [ ] **Step 2: Write the failing test**

`internal/canonical/load_test.go`:

```go
package canonical

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
    b, err := Load("testdata/sample")
    require.NoError(t, err)

    assert.Equal(t, "home", b.Config.ActiveProfile)
    assert.Equal(t, []string{"claude-code", "crush"}, b.Config.EnabledHarnesses)

    assert.Equal(t, "home", b.Profile.Name)
    assert.Equal(t, "https://gw.lan", b.Profile.Gateway.URL)

    require.Len(t, b.MCP.Servers, 1)
    assert.Equal(t, "filepuff", b.MCP.Servers[0].Name)

    require.Len(t, b.Skills, 1)
    assert.Equal(t, "hello", b.Skills[0].Name)

    require.Len(t, b.Agents, 1)
    assert.Equal(t, "sample", b.Agents[0].Name)

    assert.Contains(t, b.Instructions.Global, "Be helpful.")
}

func TestLoadMissingProfile(t *testing.T) {
    _, err := Load("testdata/missing")
    assert.Error(t, err)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/canonical/...`
Expected: FAIL with `undefined: Load`

- [ ] **Step 4: Implement loader**

`internal/canonical/load.go`:

```go
package canonical

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/goccy/go-yaml"
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

    if err := loadYAML(filepath.Join(root, "config.yaml"), &b.Config); err != nil {
        return nil, err
    }
    if b.Config.ActiveProfile == "" {
        return nil, fmt.Errorf("config.yaml: active_profile is required")
    }

    profPath := filepath.Join(root, "profiles", b.Config.ActiveProfile+".yaml")
    if err := loadYAML(profPath, &b.Profile); err != nil {
        return nil, err
    }

    mcpPath := filepath.Join(root, "mcp.yaml")
    if fileExists(mcpPath) {
        if err := loadYAML(mcpPath, &b.MCP); err != nil {
            return nil, err
        }
    }

    skills, err := loadMarkdownDir(filepath.Join(root, "skills"), "SKILL.md")
    if err != nil {
        return nil, err
    }
    for _, m := range skills {
        b.Skills = append(b.Skills, Skill{
            Name:        m.Name,
            Description: m.Description,
            Body:        m.Body,
            Path:        m.Path,
        })
    }

    agents, err := loadMarkdownDir(filepath.Join(root, "agents"), "")
    if err != nil {
        return nil, err
    }
    for _, m := range agents {
        b.Agents = append(b.Agents, Agent{
            Name:        m.Name,
            Description: m.Description,
            Body:        m.Body,
            Path:        m.Path,
        })
    }

    b.Instructions.PerHarness = map[string]string{}
    if body, err := readFileIfExists(filepath.Join(root, "instructions", "global.md")); err != nil {
        return nil, err
    } else {
        b.Instructions.Global = body
    }

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
            name = strings.TrimSuffix(filepath.Base(filepath.Dir(rel)+"/"+filepath.Base(rel)), ".md")
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
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/canonical/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/canonical
git commit -m "feat(canonical): load YAML/markdown into Bundle"
```

---

### Task 4: Secrets / env-var substitution

**Files:**
- Create: `internal/secrets/envsub.go`
- Create: `internal/secrets/envsub_test.go`

- [ ] **Step 1: Write the failing test**

`internal/secrets/envsub_test.go`:

```go
package secrets

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestSubstituteSimple(t *testing.T) {
    lookup := func(k string) (string, bool) {
        if k == "FOO" {
            return "bar", true
        }
        return "", false
    }
    out, err := Substitute("hello ${FOO}", lookup)
    assert.NoError(t, err)
    assert.Equal(t, "hello bar", out)
}

func TestSubstituteMissingStrict(t *testing.T) {
    lookup := func(k string) (string, bool) { return "", false }
    _, err := Substitute("${MISSING}", lookup)
    assert.Error(t, err)
}

func TestSubstituteEscaped(t *testing.T) {
    lookup := func(k string) (string, bool) { return "should-not-see", true }
    out, err := Substitute("$${LITERAL}", lookup)
    assert.NoError(t, err)
    assert.Equal(t, "${LITERAL}", out)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/secrets/...`
Expected: FAIL with `undefined: Substitute`

- [ ] **Step 3: Implement substitution**

`internal/secrets/envsub.go`:

```go
package secrets

import (
    "fmt"
    "os"
    "strings"
)

// Lookup resolves a placeholder name to a value. Returns false if missing.
type Lookup func(name string) (string, bool)

// OSEnv resolves via os.LookupEnv.
func OSEnv(name string) (string, bool) {
    return os.LookupEnv(name)
}

// Substitute replaces ${NAME} placeholders in s using lookup. Escape with $$.
// Missing keys produce an error so secrets are never silently empty.
func Substitute(s string, lookup Lookup) (string, error) {
    var b strings.Builder
    i := 0
    for i < len(s) {
        if i+1 < len(s) && s[i] == '$' && s[i+1] == '$' {
            b.WriteByte('$')
            i += 2
            continue
        }
        if i+1 < len(s) && s[i] == '$' && s[i+1] == '{' {
            end := strings.IndexByte(s[i+2:], '}')
            if end == -1 {
                return "", fmt.Errorf("unterminated placeholder at offset %d", i)
            }
            name := s[i+2 : i+2+end]
            val, ok := lookup(name)
            if !ok {
                return "", fmt.Errorf("missing env var %q", name)
            }
            b.WriteString(val)
            i += 2 + end + 1
            continue
        }
        b.WriteByte(s[i])
        i++
    }
    return b.String(), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/secrets/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/secrets
git commit -m "feat(secrets): ${VAR} substitution with strict missing-key error"
```

---

## Phase 2: Adapter framework

### Task 5: Adapter interface + FileSet

**Files:**
- Create: `internal/adapter/adapter.go`
- Create: `internal/adapter/fileset.go`
- Create: `internal/adapter/adapter_test.go`

- [ ] **Step 1: Write the failing test**

`internal/adapter/adapter_test.go`:

```go
package adapter

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestFileSetAddAndIterate(t *testing.T) {
    fs := NewFileSet()
    fs.Add(File{Dest: "/tmp/a", Kind: RenderedFile, Content: []byte("a")})
    fs.Add(File{Dest: "/tmp/b", Kind: SymlinkFile, SymlinkTarget: "/canon/b"})

    seen := map[string]Kind{}
    fs.ForEach(func(f File) {
        seen[f.Dest] = f.Kind
    })
    assert.Equal(t, RenderedFile, seen["/tmp/a"])
    assert.Equal(t, SymlinkFile, seen["/tmp/b"])
}

func TestKindString(t *testing.T) {
    assert.Equal(t, "rendered", RenderedFile.String())
    assert.Equal(t, "symlink-file", SymlinkFile.String())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/...`
Expected: FAIL with `undefined: NewFileSet`

- [ ] **Step 3: Implement adapter interface + fileset**

`internal/adapter/adapter.go`:

```go
package adapter

import "github.com/lukaszraczylo/harness-sync/internal/canonical"

type Kind int

const (
    RenderedFile Kind = iota
    SymlinkFile
    SymlinkDir
)

func (k Kind) String() string {
    switch k {
    case RenderedFile:
        return "rendered"
    case SymlinkFile:
        return "symlink-file"
    case SymlinkDir:
        return "symlink-dir"
    }
    return "unknown"
}

type Adapter interface {
    Name() string
    Detect() bool
    Render(b *canonical.Bundle) (*FileSet, error)
    Import(home string) (*ImportResult, error)
}

type ImportResult struct {
    Skills       []canonical.Skill
    Agents       []canonical.Agent
    MCP          []canonical.MCPServer
    Profiles     []canonical.Profile
    Instructions string // raw instruction file body (e.g. CLAUDE.md)
}
```

`internal/adapter/fileset.go`:

```go
package adapter

import "os"

type File struct {
    Dest          string
    Kind          Kind
    Content       []byte
    SymlinkTarget string
    Mode          os.FileMode
}

type FileSet struct {
    files []File
}

func NewFileSet() *FileSet { return &FileSet{} }

func (fs *FileSet) Add(f File) {
    if f.Mode == 0 && f.Kind == RenderedFile {
        f.Mode = 0o644
    }
    fs.files = append(fs.files, f)
}

func (fs *FileSet) ForEach(fn func(File)) {
    for _, f := range fs.files {
        fn(f)
    }
}

func (fs *FileSet) Len() int { return len(fs.files) }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapter/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapter
git commit -m "feat(adapter): define Adapter interface and FileSet"
```

---

### Task 6: Adapter registry

**Files:**
- Create: `internal/adapter/registry.go`
- Modify: `internal/adapter/adapter_test.go` (extend)

- [ ] **Step 1: Extend tests with a fake adapter**

Append to `internal/adapter/adapter_test.go`:

```go
type fakeAdapter struct{ name string; detect bool }

func (f *fakeAdapter) Name() string                                  { return f.name }
func (f *fakeAdapter) Detect() bool                                  { return f.detect }
func (f *fakeAdapter) Render(_ *canonical.Bundle) (*FileSet, error)  { return NewFileSet(), nil }
func (f *fakeAdapter) Import(_ string) (*ImportResult, error)        { return &ImportResult{}, nil }

func TestRegistry(t *testing.T) {
    r := NewRegistry()
    a := &fakeAdapter{name: "a", detect: true}
    b := &fakeAdapter{name: "b", detect: false}
    r.Register(a)
    r.Register(b)

    assert.Equal(t, []string{"a", "b"}, r.Names())
    assert.Equal(t, []string{"a"}, r.DetectedNames())

    got, ok := r.Get("a")
    assert.True(t, ok)
    assert.Equal(t, "a", got.Name())

    _, ok = r.Get("missing")
    assert.False(t, ok)
}

func TestRegistryDuplicatePanics(t *testing.T) {
    r := NewRegistry()
    r.Register(&fakeAdapter{name: "x"})
    assert.Panics(t, func() { r.Register(&fakeAdapter{name: "x"}) })
}
```

Add `import "github.com/lukaszraczylo/harness-sync/internal/canonical"` to test imports if not present.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/...`
Expected: FAIL with `undefined: NewRegistry`

- [ ] **Step 3: Implement registry**

`internal/adapter/registry.go`:

```go
package adapter

import "fmt"

type Registry struct {
    order []string
    byName map[string]Adapter
}

func NewRegistry() *Registry {
    return &Registry{byName: map[string]Adapter{}}
}

func (r *Registry) Register(a Adapter) {
    if _, exists := r.byName[a.Name()]; exists {
        panic(fmt.Sprintf("adapter %q already registered", a.Name()))
    }
    r.byName[a.Name()] = a
    r.order = append(r.order, a.Name())
}

func (r *Registry) Names() []string {
    out := make([]string, len(r.order))
    copy(out, r.order)
    return out
}

func (r *Registry) DetectedNames() []string {
    var out []string
    for _, n := range r.order {
        if r.byName[n].Detect() {
            out = append(out, n)
        }
    }
    return out
}

func (r *Registry) Get(name string) (Adapter, bool) {
    a, ok := r.byName[name]
    return a, ok
}

func (r *Registry) All() []Adapter {
    out := make([]Adapter, 0, len(r.order))
    for _, n := range r.order {
        out = append(out, r.byName[n])
    }
    return out
}

// Default is the process-global registry used by main(). Tests should
// construct their own Registry via NewRegistry to avoid global state.
var Default = NewRegistry()
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapter/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapter
git commit -m "feat(adapter): registry with detect filtering and duplicate guard"
```

---

## Phase 3: Render, git, merge

### Task 7: Render helpers (JSON / YAML / TOML)

**Files:**
- Create: `internal/render/json.go`
- Create: `internal/render/yaml.go`
- Create: `internal/render/toml.go`
- Create: `internal/render/render_test.go`

- [ ] **Step 1: Write the failing test**

`internal/render/render_test.go`:

```go
package render

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestJSONIndent(t *testing.T) {
    b, err := JSON(map[string]any{"a": 1, "b": "x"})
    require.NoError(t, err)
    s := string(b)
    assert.Contains(t, s, "\"a\": 1")
    assert.True(t, s[len(s)-1] == '\n', "trailing newline required")
}

func TestYAML(t *testing.T) {
    b, err := YAML(map[string]any{"a": 1})
    require.NoError(t, err)
    assert.Contains(t, string(b), "a: 1")
}

func TestTOML(t *testing.T) {
    b, err := TOML(map[string]any{"a": 1})
    require.NoError(t, err)
    assert.Contains(t, string(b), "a = 1")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/...`
Expected: FAIL with `undefined: JSON`

- [ ] **Step 3: Implement renderers**

`internal/render/json.go`:

```go
package render

import (
    "bytes"
    "encoding/json"
)

func JSON(v any) ([]byte, error) {
    var buf bytes.Buffer
    enc := json.NewEncoder(&buf)
    enc.SetIndent("", "  ")
    enc.SetEscapeHTML(false)
    if err := enc.Encode(v); err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}
```

`internal/render/yaml.go`:

```go
package render

import "github.com/goccy/go-yaml"

func YAML(v any) ([]byte, error) {
    return yaml.MarshalWithOptions(v, yaml.Indent(2))
}
```

`internal/render/toml.go`:

```go
package render

import "github.com/pelletier/go-toml/v2"

func TOML(v any) ([]byte, error) {
    return toml.Marshal(v)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/render/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/render
git commit -m "feat(render): JSON/YAML/TOML marshallers with deterministic formatting"
```

---

### Task 8: Git CLI wrapper

**Files:**
- Create: `internal/gitx/repo.go`
- Create: `internal/gitx/repo_test.go`

- [ ] **Step 1: Write the failing test**

`internal/gitx/repo_test.go`:

```go
package gitx

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestInitAddCommit(t *testing.T) {
    dir := t.TempDir()
    r := New(dir)

    require.NoError(t, r.Init())
    require.NoError(t, r.Configure("nobody", "nobody@example.com"))

    require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi"), 0o644))
    require.NoError(t, r.AddAll())
    require.NoError(t, r.Commit("initial"))

    head, err := r.HeadCommit()
    require.NoError(t, err)
    assert.NotEmpty(t, head)
}

func TestShowFileAtHead(t *testing.T) {
    dir := t.TempDir()
    r := New(dir)
    require.NoError(t, r.Init())
    require.NoError(t, r.Configure("nobody", "nobody@example.com"))
    require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("v1"), 0o644))
    require.NoError(t, r.AddAll())
    require.NoError(t, r.Commit("v1"))

    body, err := r.ShowFileAtHead("a.txt")
    require.NoError(t, err)
    assert.Equal(t, "v1", string(body))

    _, err = r.ShowFileAtHead("nope.txt")
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/gitx/...`
Expected: FAIL with `undefined: New`

- [ ] **Step 3: Implement git wrapper**

`internal/gitx/repo.go`:

```go
package gitx

import (
    "bytes"
    "fmt"
    "os/exec"
)

type Repo struct {
    Dir string
}

func New(dir string) *Repo { return &Repo{Dir: dir} }

func (r *Repo) run(args ...string) ([]byte, error) {
    cmd := exec.Command("git", args...)
    cmd.Dir = r.Dir
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("git %v: %w: %s", args, err, stderr.String())
    }
    return stdout.Bytes(), nil
}

func (r *Repo) Init() error {
    _, err := r.run("init", "-q", "-b", "main")
    return err
}

func (r *Repo) Configure(name, email string) error {
    if _, err := r.run("config", "user.name", name); err != nil {
        return err
    }
    _, err := r.run("config", "user.email", email)
    return err
}

func (r *Repo) AddAll() error {
    _, err := r.run("add", "-A")
    return err
}

func (r *Repo) Add(paths ...string) error {
    args := append([]string{"add", "--"}, paths...)
    _, err := r.run(args...)
    return err
}

func (r *Repo) Commit(msg string) error {
    _, err := r.run("commit", "-q", "-m", msg)
    return err
}

func (r *Repo) HeadCommit() (string, error) {
    b, err := r.run("rev-parse", "HEAD")
    return string(bytes.TrimSpace(b)), err
}

func (r *Repo) ShowFileAtHead(path string) ([]byte, error) {
    return r.run("show", "HEAD:"+path)
}

func (r *Repo) IsRepo() bool {
    _, err := r.run("rev-parse", "--git-dir")
    return err == nil
}

func (r *Repo) Revert(n int) error {
    _, err := r.run("revert", "--no-edit", fmt.Sprintf("HEAD~%d..HEAD", n-1))
    return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/gitx/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gitx
git commit -m "feat(gitx): thin shell wrapper over git CLI for repo ops"
```

---

### Task 9: Three-way merge

**Files:**
- Create: `internal/merge/merge.go`
- Create: `internal/merge/merge_test.go`

- [ ] **Step 1: Write the failing test**

`internal/merge/merge_test.go`:

```go
package merge

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCleanFastForward(t *testing.T) {
    res, err := ThreeWay(Inputs{
        Base:    []byte("a\nb\nc\n"),
        Ours:    []byte("a\nb\nc\nd\n"),
        Theirs:  []byte("a\nb\nc\n"),
    })
    require.NoError(t, err)
    assert.False(t, res.Conflict)
    assert.Equal(t, "a\nb\nc\nd\n", string(res.Body))
}

func TestConflict(t *testing.T) {
    res, err := ThreeWay(Inputs{
        Base:    []byte("a\n"),
        Ours:    []byte("a\nours\n"),
        Theirs:  []byte("a\ntheirs\n"),
    })
    require.NoError(t, err)
    assert.True(t, res.Conflict)
    assert.Contains(t, string(res.Body), "<<<<<<<")
    assert.Contains(t, string(res.Body), ">>>>>>>")
}

func TestNoOpWhenAllEqual(t *testing.T) {
    res, err := ThreeWay(Inputs{
        Base:    []byte("a\n"),
        Ours:    []byte("a\n"),
        Theirs:  []byte("a\n"),
    })
    require.NoError(t, err)
    assert.False(t, res.Conflict)
    assert.Equal(t, "a\n", string(res.Body))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/merge/...`
Expected: FAIL with `undefined: ThreeWay`

- [ ] **Step 3: Implement merge wrapper**

`internal/merge/merge.go`:

```go
package merge

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

// Inputs:
//   Base   = last-rendered snapshot (state/...)
//   Ours   = new rendered output (what we'd write now)
//   Theirs = current target file contents (what's on disk at the destination)
type Inputs struct {
    Base   []byte
    Ours   []byte
    Theirs []byte
}

type Result struct {
    Body     []byte
    Conflict bool
}

// ThreeWay runs `git merge-file --diff3 ours base theirs` and returns the
// merged body. When Conflict is true, Body contains conflict markers.
func ThreeWay(in Inputs) (*Result, error) {
    tmp, err := os.MkdirTemp("", "harness-sync-merge-")
    if err != nil {
        return nil, err
    }
    defer os.RemoveAll(tmp)

    oursPath := filepath.Join(tmp, "ours")
    basePath := filepath.Join(tmp, "base")
    theirsPath := filepath.Join(tmp, "theirs")

    if err := os.WriteFile(oursPath, in.Ours, 0o644); err != nil {
        return nil, err
    }
    if err := os.WriteFile(basePath, in.Base, 0o644); err != nil {
        return nil, err
    }
    if err := os.WriteFile(theirsPath, in.Theirs, 0o644); err != nil {
        return nil, err
    }

    cmd := exec.Command(
        "git", "merge-file",
        "-L", "harness-sync (new)",
        "-L", "harness-sync (base)",
        "-L", "harness-sync (current)",
        "-p",
        oursPath, basePath, theirsPath,
    )
    out, runErr := cmd.Output()

    res := &Result{Body: out}
    if runErr != nil {
        if ee, ok := runErr.(*exec.ExitError); ok {
            // git merge-file exits with conflict-count on conflict, body still valid
            if ee.ExitCode() > 0 {
                res.Conflict = true
                return res, nil
            }
        }
        return nil, fmt.Errorf("git merge-file: %w", runErr)
    }
    return res, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/merge/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/merge
git commit -m "feat(merge): three-way merge via git merge-file with conflict detection"
```

---

## Phase 4: Apply pipeline + first adapter

### Task 10: Apply pipeline

**Files:**
- Create: `internal/apply/apply.go`
- Create: `internal/apply/apply_test.go`

The apply pipeline takes a Bundle + list of adapters and writes their rendered FileSets to disk, recording state and reporting conflicts.

- [ ] **Step 1: Write the failing test**

`internal/apply/apply_test.go`:

```go
package apply

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/canonical"
    "github.com/lukaszraczylo/harness-sync/internal/gitx"
)

type stubAdapter struct {
    name  string
    files []adapter.File
}

func (s *stubAdapter) Name() string                                       { return s.name }
func (s *stubAdapter) Detect() bool                                       { return true }
func (s *stubAdapter) Import(_ string) (*adapter.ImportResult, error)     { return &adapter.ImportResult{}, nil }
func (s *stubAdapter) Render(_ *canonical.Bundle) (*adapter.FileSet, error) {
    fs := adapter.NewFileSet()
    for _, f := range s.files {
        fs.Add(f)
    }
    return fs, nil
}

func TestApplyRenderedFileFreshWrite(t *testing.T) {
    root := t.TempDir()
    target := filepath.Join(t.TempDir(), "out.json")

    require.NoError(t, initCanonical(root))

    ad := &stubAdapter{
        name: "stub",
        files: []adapter.File{
            {Dest: target, Kind: adapter.RenderedFile, Content: []byte("{}\n")},
        },
    }
    rep, err := Run(Options{
        Bundle:   &canonical.Bundle{Root: root},
        Adapters: []adapter.Adapter{ad},
        Repo:     gitx.New(root),
    })
    require.NoError(t, err)
    assert.Equal(t, 1, rep.Written)
    assert.Equal(t, 0, rep.Conflicts)

    body, err := os.ReadFile(target)
    require.NoError(t, err)
    assert.Equal(t, "{}\n", string(body))

    // state snapshot recorded
    statePath := filepath.Join(root, "state", "stub", target)
    assert.FileExists(t, statePath)
}

func TestApplyConflictWritesRej(t *testing.T) {
    root := t.TempDir()
    target := filepath.Join(t.TempDir(), "out.txt")

    require.NoError(t, initCanonical(root))
    repo := gitx.New(root)

    // First apply
    ad1 := &stubAdapter{
        name: "stub",
        files: []adapter.File{
            {Dest: target, Kind: adapter.RenderedFile, Content: []byte("base\n")},
        },
    }
    _, err := Run(Options{
        Bundle:   &canonical.Bundle{Root: root},
        Adapters: []adapter.Adapter{ad1},
        Repo:     repo,
    })
    require.NoError(t, err)

    // User edits target out-of-band
    require.NoError(t, os.WriteFile(target, []byte("user-edit\n"), 0o644))

    // Second apply changes the rendered content too
    ad2 := &stubAdapter{
        name: "stub",
        files: []adapter.File{
            {Dest: target, Kind: adapter.RenderedFile, Content: []byte("new\n")},
        },
    }
    rep, err := Run(Options{
        Bundle:   &canonical.Bundle{Root: root},
        Adapters: []adapter.Adapter{ad2},
        Repo:     repo,
    })
    require.NoError(t, err)
    assert.Equal(t, 1, rep.Conflicts)
    assert.FileExists(t, target+".rej")
}

func initCanonical(root string) error {
    if err := os.MkdirAll(filepath.Join(root, "state"), 0o755); err != nil {
        return err
    }
    r := gitx.New(root)
    if err := r.Init(); err != nil {
        return err
    }
    if err := r.Configure("test", "test@example.com"); err != nil {
        return err
    }
    if err := os.WriteFile(filepath.Join(root, ".gitkeep"), []byte{}, 0o644); err != nil {
        return err
    }
    if err := r.AddAll(); err != nil {
        return err
    }
    return r.Commit("init")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/apply/...`
Expected: FAIL with `undefined: Run`

- [ ] **Step 3: Implement apply pipeline**

`internal/apply/apply.go`:

```go
package apply

import (
    "errors"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/canonical"
    "github.com/lukaszraczylo/harness-sync/internal/gitx"
    "github.com/lukaszraczylo/harness-sync/internal/merge"
)

type Options struct {
    Bundle   *canonical.Bundle
    Adapters []adapter.Adapter
    Repo     *gitx.Repo
    DryRun   bool
    Force    bool
}

type Report struct {
    Written   int
    Skipped   int
    Conflicts int
    Actions   []Action
}

type Action struct {
    Adapter string
    Dest    string
    Kind    string // "wrote" | "symlinked" | "skipped" | "conflict"
    Note    string
}

func Run(opt Options) (*Report, error) {
    rep := &Report{}
    for _, ad := range opt.Adapters {
        fs, err := ad.Render(opt.Bundle)
        if err != nil {
            return rep, fmt.Errorf("render %s: %w", ad.Name(), err)
        }
        var renderErr error
        fs.ForEach(func(f adapter.File) {
            if renderErr != nil {
                return
            }
            renderErr = handle(opt, ad.Name(), f, rep)
        })
        if renderErr != nil {
            return rep, renderErr
        }
    }
    if !opt.DryRun && rep.Written > 0 && opt.Repo != nil {
        if err := opt.Repo.AddAll(); err != nil {
            return rep, err
        }
        if err := opt.Repo.Commit(fmt.Sprintf("apply: %d files, %d conflicts", rep.Written, rep.Conflicts)); err != nil {
            return rep, err
        }
    }
    return rep, nil
}

func handle(opt Options, adapterName string, f adapter.File, rep *Report) error {
    switch f.Kind {
    case adapter.RenderedFile:
        return handleRendered(opt, adapterName, f, rep)
    case adapter.SymlinkFile, adapter.SymlinkDir:
        return handleSymlink(opt, adapterName, f, rep)
    }
    return fmt.Errorf("unknown file kind for %s", f.Dest)
}

func statePath(root, adapterName, dest string) string {
    return filepath.Join(root, "state", adapterName, dest)
}

func handleRendered(opt Options, adapterName string, f adapter.File, rep *Report) error {
    sp := statePath(opt.Bundle.Root, adapterName, f.Dest)
    base, _ := os.ReadFile(sp) // empty if missing
    current, _ := os.ReadFile(f.Dest)

    if string(current) == string(f.Content) {
        rep.Skipped++
        rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "skipped", Note: "already in sync"})
        return writeState(opt, sp, f.Content)
    }

    if opt.Force || len(base) == 0 || string(current) == string(base) {
        if opt.DryRun {
            rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "wrote", Note: "would write"})
            return nil
        }
        if err := writeFile(f.Dest, f.Content, f.Mode); err != nil {
            return err
        }
        if err := writeState(opt, sp, f.Content); err != nil {
            return err
        }
        rep.Written++
        rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "wrote"})
        return nil
    }

    res, err := merge.ThreeWay(merge.Inputs{Base: base, Ours: f.Content, Theirs: current})
    if err != nil {
        return err
    }
    if res.Conflict {
        if opt.DryRun {
            rep.Conflicts++
            rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "conflict", Note: "would write .rej"})
            return nil
        }
        if err := os.WriteFile(f.Dest+".rej", res.Body, 0o644); err != nil {
            return err
        }
        rep.Conflicts++
        rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "conflict", Note: "wrote .rej"})
        return nil
    }
    if opt.DryRun {
        rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "wrote", Note: "would merge"})
        return nil
    }
    if err := writeFile(f.Dest, res.Body, f.Mode); err != nil {
        return err
    }
    if err := writeState(opt, sp, f.Content); err != nil {
        return err
    }
    rep.Written++
    rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "wrote", Note: "merged"})
    return nil
}

func handleSymlink(opt Options, adapterName string, f adapter.File, rep *Report) error {
    existing, err := os.Readlink(f.Dest)
    if err == nil && existing == f.SymlinkTarget {
        rep.Skipped++
        rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "skipped", Note: "symlink already correct"})
        return nil
    }
    if opt.DryRun {
        rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "symlinked", Note: "would link"})
        return nil
    }
    if _, statErr := os.Lstat(f.Dest); statErr == nil {
        // back up existing
        backup := filepath.Join(opt.Bundle.Root, "backups", adapterName, filepath.Base(f.Dest))
        if err := os.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
            return err
        }
        if err := os.Rename(f.Dest, backup); err != nil && !errors.Is(err, fs.ErrNotExist) {
            return err
        }
    }
    if err := os.MkdirAll(filepath.Dir(f.Dest), 0o755); err != nil {
        return err
    }
    if err := os.Symlink(f.SymlinkTarget, f.Dest); err != nil {
        return err
    }
    rep.Written++
    rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "symlinked"})
    return nil
}

func writeFile(dest string, body []byte, mode os.FileMode) error {
    if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
        return err
    }
    tmp := dest + ".tmp"
    if mode == 0 {
        mode = 0o644
    }
    if err := os.WriteFile(tmp, body, mode); err != nil {
        return err
    }
    return os.Rename(tmp, dest)
}

func writeState(opt Options, statePath string, body []byte) error {
    if opt.DryRun {
        return nil
    }
    if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
        return err
    }
    return os.WriteFile(statePath, body, 0o644)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/apply/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/apply
git commit -m "feat(apply): render+merge+write pipeline with state snapshots and .rej conflicts"
```

---

### Task 11: claudecode adapter (Detect + Render)

**Files:**
- Create: `internal/adapters/claudecode/claudecode.go`
- Create: `internal/adapters/claudecode/claudecode_test.go`
- Create: `internal/adapters/claudecode/testdata/`

For claude-code, canonical skills/agents map 1:1 to `~/.claude/skills/` and `~/.claude/agents/`. Global instructions → `~/.claude/CLAUDE.md`. MCP block lives in `~/.claude/settings.json`.

- [ ] **Step 1: Inspect actual claude-code layout to ground the adapter**

```bash
ls ~/.claude/
ls ~/.claude/skills/ | head
ls ~/.claude/agents/ | head
test -f ~/.claude/settings.json && head -c 2000 ~/.claude/settings.json
```

Note: the adapter only needs to produce files in known locations. Settings.json shape: add `mcpServers` key with name → `{command, args, env, transport}`.

- [ ] **Step 2: Write the failing test**

`internal/adapters/claudecode/claudecode_test.go`:

```go
package claudecode

import (
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestRenderProducesExpectedTargets(t *testing.T) {
    home := t.TempDir()
    ad := New(WithHome(home))
    b := &canonical.Bundle{
        Root: "/canon",
        Config: canonical.Config{ActiveProfile: "home"},
        MCP: canonical.MCPRegistry{Servers: []canonical.MCPServer{
            {Name: "filepuff", Command: "/bin/filepuff", Transport: "stdio"},
        }},
        Instructions: canonical.Instructions{Global: "# global"},
    }
    fs, err := ad.Render(b)
    require.NoError(t, err)

    seen := map[string]adapter.File{}
    fs.ForEach(func(f adapter.File) { seen[f.Dest] = f })

    skillsDest := filepath.Join(home, ".claude", "skills")
    assert.Equal(t, adapter.SymlinkDir, seen[skillsDest].Kind)
    assert.Equal(t, "/canon/skills", seen[skillsDest].SymlinkTarget)

    agentsDest := filepath.Join(home, ".claude", "agents")
    assert.Equal(t, adapter.SymlinkDir, seen[agentsDest].Kind)
    assert.Equal(t, "/canon/agents", seen[agentsDest].SymlinkTarget)

    claudemdDest := filepath.Join(home, ".claude", "CLAUDE.md")
    assert.Equal(t, adapter.RenderedFile, seen[claudemdDest].Kind)
    assert.Contains(t, string(seen[claudemdDest].Content), "# global")

    settingsDest := filepath.Join(home, ".claude", "settings.json")
    assert.Equal(t, adapter.RenderedFile, seen[settingsDest].Kind)
    assert.Contains(t, string(seen[settingsDest].Content), "\"filepuff\"")
    assert.Contains(t, string(seen[settingsDest].Content), "\"command\": \"/bin/filepuff\"")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/claudecode/...`
Expected: FAIL with `undefined: New`

- [ ] **Step 4: Implement adapter**

`internal/adapters/claudecode/claudecode.go`:

```go
package claudecode

import (
    "os"
    "path/filepath"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/canonical"
    "github.com/lukaszraczylo/harness-sync/internal/render"
)

const name = "claude-code"

type Adapter struct {
    home string
}

type Option func(*Adapter)

func WithHome(h string) Option { return func(a *Adapter) { a.home = h } }

func New(opts ...Option) *Adapter {
    a := &Adapter{home: defaultHome()}
    for _, o := range opts {
        o(a)
    }
    return a
}

func defaultHome() string {
    h, err := os.UserHomeDir()
    if err != nil {
        return ""
    }
    return h
}

func (a *Adapter) Name() string { return name }

func (a *Adapter) Detect() bool {
    _, err := os.Stat(filepath.Join(a.home, ".claude"))
    return err == nil
}

func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
    fs := adapter.NewFileSet()
    base := filepath.Join(a.home, ".claude")

    fs.Add(adapter.File{
        Dest:          filepath.Join(base, "skills"),
        Kind:          adapter.SymlinkDir,
        SymlinkTarget: filepath.Join(b.Root, "skills"),
    })
    fs.Add(adapter.File{
        Dest:          filepath.Join(base, "agents"),
        Kind:          adapter.SymlinkDir,
        SymlinkTarget: filepath.Join(b.Root, "agents"),
    })

    instructions := b.Instructions.Global
    if override, ok := b.Instructions.PerHarness[name]; ok && override != "" {
        instructions = override
    }
    fs.Add(adapter.File{
        Dest:    filepath.Join(base, "CLAUDE.md"),
        Kind:    adapter.RenderedFile,
        Content: []byte(instructions),
    })

    settings, err := renderSettings(b)
    if err != nil {
        return nil, err
    }
    fs.Add(adapter.File{
        Dest:    filepath.Join(base, "settings.json"),
        Kind:    adapter.RenderedFile,
        Content: settings,
    })

    return fs, nil
}

func renderSettings(b *canonical.Bundle) ([]byte, error) {
    mcp := map[string]any{}
    for _, s := range b.MCP.Servers {
        entry := map[string]any{}
        if s.Command != "" {
            entry["command"] = s.Command
        }
        if len(s.Args) > 0 {
            entry["args"] = s.Args
        }
        if s.URL != "" {
            entry["url"] = s.URL
        }
        if s.Transport != "" {
            entry["transport"] = s.Transport
        }
        if len(s.Env) > 0 {
            entry["env"] = s.Env
        }
        mcp[s.Name] = entry
    }
    return render.JSON(map[string]any{
        "mcpServers": mcp,
    })
}

func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
    return importFrom(home)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/adapters/claudecode/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/adapters/claudecode
git commit -m "feat(claudecode): adapter with skills+agents symlinks, CLAUDE.md, settings.json"
```

---

### Task 12: claudecode adapter Import

**Files:**
- Create: `internal/adapters/claudecode/import.go`
- Modify: `internal/adapters/claudecode/claudecode_test.go`

- [ ] **Step 1: Extend tests with an Import case**

Append to `internal/adapters/claudecode/claudecode_test.go`:

```go
import (
    "encoding/json"
    "os"
)

func TestImport(t *testing.T) {
    home := t.TempDir()
    base := filepath.Join(home, ".claude")
    require.NoError(t, os.MkdirAll(filepath.Join(base, "skills", "hello"), 0o755))
    require.NoError(t, os.MkdirAll(filepath.Join(base, "agents"), 0o755))

    require.NoError(t, os.WriteFile(
        filepath.Join(base, "skills", "hello", "SKILL.md"),
        []byte("---\nname: hello\ndescription: x\n---\nbody"), 0o644))
    require.NoError(t, os.WriteFile(
        filepath.Join(base, "agents", "agentA.md"),
        []byte("---\nname: agentA\ndescription: y\n---\nagent body"), 0o644))
    require.NoError(t, os.WriteFile(
        filepath.Join(base, "CLAUDE.md"),
        []byte("# global"), 0o644))

    settings, _ := json.Marshal(map[string]any{
        "mcpServers": map[string]any{
            "filepuff": map[string]any{
                "command":   "/bin/filepuff",
                "transport": "stdio",
            },
        },
    })
    require.NoError(t, os.WriteFile(
        filepath.Join(base, "settings.json"), settings, 0o644))

    ad := New(WithHome(home))
    res, err := ad.Import(home)
    require.NoError(t, err)

    require.Len(t, res.Skills, 1)
    assert.Equal(t, "hello", res.Skills[0].Name)
    require.Len(t, res.Agents, 1)
    assert.Equal(t, "agentA", res.Agents[0].Name)
    require.Len(t, res.MCP, 1)
    assert.Equal(t, "filepuff", res.MCP[0].Name)
    assert.Equal(t, "/bin/filepuff", res.MCP[0].Command)
    assert.Equal(t, "# global", res.Instructions)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/claudecode/...`
Expected: FAIL with `undefined: importFrom`

- [ ] **Step 3: Implement Import**

`internal/adapters/claudecode/import.go`:

```go
package claudecode

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"

    "github.com/goccy/go-yaml"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func importFrom(home string) (*adapter.ImportResult, error) {
    base := filepath.Join(home, ".claude")
    res := &adapter.ImportResult{}

    if skills, err := importMarkdownTree(filepath.Join(base, "skills"), "SKILL.md"); err != nil {
        return nil, err
    } else {
        for _, s := range skills {
            res.Skills = append(res.Skills, canonical.Skill(s))
        }
    }
    if agents, err := importMarkdownTree(filepath.Join(base, "agents"), ""); err != nil {
        return nil, err
    } else {
        for _, a := range agents {
            res.Agents = append(res.Agents, canonical.Agent(a))
        }
    }
    if body, err := readIfExists(filepath.Join(base, "CLAUDE.md")); err != nil {
        return nil, err
    } else {
        res.Instructions = body
    }
    if servers, err := importMCPFromSettings(filepath.Join(base, "settings.json")); err != nil {
        return nil, err
    } else {
        res.MCP = servers
    }
    return res, nil
}

type doc struct {
    Name        string
    Description string
    Body        string
    Path        string
}

func importMarkdownTree(dir, requiredFilename string) ([]doc, error) {
    var docs []doc
    if !dirExists(dir) {
        return docs, nil
    }
    return docs, filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
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
        name, desc := parseFrontmatter(body)
        if name == "" {
            name = strings.TrimSuffix(filepath.Base(p), ".md")
        }
        rel, _ := filepath.Rel(dir, p)
        docs = append(docs, doc{
            Name:        name,
            Description: desc,
            Body:        string(body),
            Path:        rel,
        })
        return nil
    })
}

func importMCPFromSettings(path string) ([]canonical.MCPServer, error) {
    body, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    var doc struct {
        MCPServers map[string]struct {
            Command   string            `json:"command"`
            Args      []string          `json:"args"`
            URL       string            `json:"url"`
            Transport string            `json:"transport"`
            Env       map[string]string `json:"env"`
        } `json:"mcpServers"`
    }
    if err := json.Unmarshal(body, &doc); err != nil {
        return nil, err
    }
    out := make([]canonical.MCPServer, 0, len(doc.MCPServers))
    for name, v := range doc.MCPServers {
        out = append(out, canonical.MCPServer{
            Name: name, Command: v.Command, Args: v.Args, URL: v.URL,
            Transport: v.Transport, Env: v.Env,
        })
    }
    return out, nil
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/claudecode/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/claudecode
git commit -m "feat(claudecode): Import reads existing ~/.claude into canonical model"
```

---

## Phase 5: Remaining adapters

Each remaining adapter follows the same shape as claudecode but with its own native schema. Before implementing, the engineer must inspect the harness's actual config files on this machine. The harness's schema is the source of truth — match its keys.

For every adapter task below, the sub-steps are identical structure (TDD). The pattern:

1. Inspect `~/.config/<harness>/` and any official docs to determine schema
2. Add the adapter package and minimum failing test
3. Implement Render
4. Implement Import
5. Register in `cmd/harness-sync/main.go`'s adapter registry init block
6. Commit

The native schema specifics are NOT prescribed by this plan; the engineer must read the harness's actual files in `~/.config/<harness>/` and any official docs and faithfully reproduce them. If a harness has no concept of an asset (e.g. no skill support), the adapter simply omits that Target.

### Task 13: crush adapter

**Files:**
- Create: `internal/adapters/crush/crush.go`
- Create: `internal/adapters/crush/import.go`
- Create: `internal/adapters/crush/crush_test.go`
- Create: `internal/adapters/crush/testdata/`

- [ ] **Step 1: Inspect crush's actual layout**

```bash
ls -la ~/.config/crush/
test -f ~/.config/crush/crush.json && head -c 4000 ~/.config/crush/crush.json
test -d ~/.config/crush/skills && ls ~/.config/crush/skills/
```

Document the schema you observe in a comment at the top of `crush.go`. Crush stores its config in `crush.json` and has a `prompts/` (or `skills/`) directory. Symlink the canonical skills tree if shape matches; otherwise render a flattened version.

- [ ] **Step 2: Write the failing test mirroring TestRenderProducesExpectedTargets from claudecode but with crush paths and crush.json structure**

Use the same test pattern: build a Bundle, call Render, walk the FileSet, assert each Target. Adapt destination paths to `~/.config/crush/...`.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/crush/...`
Expected: FAIL with `undefined: New`

- [ ] **Step 4: Implement Render** following the claudecode pattern. The crush LLM config block (model providers + gateway URL) must be derived from `b.Profile`. See the canonical Profile type at `internal/canonical/types.go` for fields. Use `internal/render.JSON` for crush.json output.

- [ ] **Step 5: Implement Import** by reading `~/.config/crush/crush.json` and (if present) `~/.config/crush/skills/` into `canonical.MCPServer`, `canonical.Profile`, and skill entries.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/adapters/crush/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/adapters/crush
git commit -m "feat(crush): adapter with crush.json render + skills symlink"
```

### Task 14: kilo adapter

**Files:**
- Create: `internal/adapters/kilo/kilo.go`
- Create: `internal/adapters/kilo/import.go`
- Create: `internal/adapters/kilo/kilo_test.go`

- [ ] **Step 1: Inspect kilo's actual layout**

```bash
ls -la ~/.config/kilo/
test -f ~/.config/kilo/kilo.json && head -c 4000 ~/.config/kilo/kilo.json
test -f ~/.config/kilo/opencode.jsonc && head -c 4000 ~/.config/kilo/opencode.jsonc
test -d ~/.config/kilo/agent && ls ~/.config/kilo/agent/
```

- [ ] **Step 2: Write the failing test**

Mirror claudecode TestRender pattern with kilo destinations.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/kilo/...`
Expected: FAIL with `undefined: New`

- [ ] **Step 4: Implement Render** using `internal/render.JSON`. kilo supports an `agent` directory; symlink `agents/` from canonical if shape matches, otherwise render.

- [ ] **Step 5: Implement Import** from `kilo.json` (and `opencode.jsonc` if kilo also keeps an opencode config).

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/adapters/kilo/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/adapters/kilo
git commit -m "feat(kilo): adapter with kilo.json render + agent symlink"
```

### Task 15: opencode adapter

**Files:**
- Create: `internal/adapters/opencode/opencode.go`
- Create: `internal/adapters/opencode/import.go`
- Create: `internal/adapters/opencode/opencode_test.go`

- [ ] **Step 1: Inspect opencode's actual layout**

```bash
ls -la ~/.config/opencode/
test -f ~/.config/opencode/opencode.jsonc && head -c 4000 ~/.config/opencode/opencode.jsonc
```

opencode uses JSONC. For Render, output plain JSON (JSONC parsers accept JSON). For Import, strip comments before unmarshal — use a simple line-by-line stripper for `//` comments and a regex for `/* */` blocks; do not pull in a JSONC dependency.

- [ ] **Step 2: Write the failing test** mirroring claudecode pattern with opencode destinations.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/opencode/...`
Expected: FAIL with `undefined: New`

- [ ] **Step 4: Implement Render** using `internal/render.JSON`. opencode supports `AGENTS.md` for global instructions; route `b.Instructions.Global` there.

- [ ] **Step 5: Implement Import** from `opencode.jsonc` and any existing `AGENTS.md`.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/adapters/opencode/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/adapters/opencode
git commit -m "feat(opencode): adapter with opencode.jsonc render + AGENTS.md"
```

### Task 16: goose adapter

**Files:**
- Create: `internal/adapters/goose/goose.go`
- Create: `internal/adapters/goose/import.go`
- Create: `internal/adapters/goose/goose_test.go`

- [ ] **Step 1: Inspect goose's actual layout**

```bash
ls -la ~/.config/goose/
test -f ~/.config/goose/config.yaml && head -c 4000 ~/.config/goose/config.yaml
test -d ~/.config/goose/custom_providers && ls ~/.config/goose/custom_providers/
```

goose uses YAML. Use `internal/render.YAML` for output.

- [ ] **Step 2: Write the failing test** mirroring claudecode pattern.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/goose/...`
Expected: FAIL with `undefined: New`

- [ ] **Step 4: Implement Render** for `config.yaml` + custom provider definitions derived from `b.Profile`.

- [ ] **Step 5: Implement Import** from `config.yaml`.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/adapters/goose/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/adapters/goose
git commit -m "feat(goose): adapter with config.yaml render and providers from Profile"
```

### Task 17: cagent adapter

**Files:**
- Create: `internal/adapters/cagent/cagent.go`
- Create: `internal/adapters/cagent/import.go`
- Create: `internal/adapters/cagent/cagent_test.go`

- [ ] **Step 1: Inspect cagent's actual layout**

```bash
ls -la ~/.config/cagent/
find ~/.config/cagent -type f -maxdepth 3 | head -30
```

cagent is Docker's CLI agent. Determine its config format. If `.cagent_first_run` is the only file, no config has been generated yet — write a minimal adapter that writes the expected canonical config to its expected path; consult docker docs via WebFetch if needed: `https://github.com/docker/cagent`.

- [ ] **Step 2: Write the failing test** asserting basic Render output.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/cagent/...`
Expected: FAIL with `undefined: New`

- [ ] **Step 4: Implement Render** based on observed/documented schema.

- [ ] **Step 5: Implement Import** (may be a no-op stub if cagent has no persistent config to read).

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/adapters/cagent/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/adapters/cagent
git commit -m "feat(cagent): adapter scaffold matching docker cagent layout"
```

### Task 18: zed adapter

**Files:**
- Create: `internal/adapters/zed/zed.go`
- Create: `internal/adapters/zed/import.go`
- Create: `internal/adapters/zed/zed_test.go`

- [ ] **Step 1: Inspect zed's actual layout**

```bash
ls -la ~/.config/zed/
test -f ~/.config/zed/settings.json && head -c 4000 ~/.config/zed/settings.json
```

zed is an editor; harness-sync targets its `assistant` / `language_models` settings only. Do not touch other zed settings. Use a partial-merge approach: read existing settings.json, replace only `assistant.providers` and `language_models` keys, write back. Implement this carefully because zed settings are user-edited.

- [ ] **Step 2: Write the failing test** asserting that unrelated keys are preserved.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/zed/...`
Expected: FAIL with `undefined: New`

- [ ] **Step 4: Implement Render** that loads existing settings.json (or empty object if missing), overwrites only LLM-related keys derived from `b.Profile`, marshals back via `internal/render.JSON`.

- [ ] **Step 5: Implement Import** of those same keys.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/adapters/zed/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/adapters/zed
git commit -m "feat(zed): adapter that updates only language_models keys in settings.json"
```

---

## Phase 6: CLI commands

### Task 19: Wire registry + detect command

**Files:**
- Modify: `cmd/harness-sync/main.go`
- Modify: `internal/cli/root.go`
- Create: `internal/cli/detect.go`
- Create: `internal/cli/detect_test.go`

- [ ] **Step 1: Wire all adapters into the default registry in main**

Replace `cmd/harness-sync/main.go` with:

```go
package main

import (
    "fmt"
    "os"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/adapters/cagent"
    "github.com/lukaszraczylo/harness-sync/internal/adapters/claudecode"
    "github.com/lukaszraczylo/harness-sync/internal/adapters/crush"
    "github.com/lukaszraczylo/harness-sync/internal/adapters/goose"
    "github.com/lukaszraczylo/harness-sync/internal/adapters/kilo"
    "github.com/lukaszraczylo/harness-sync/internal/adapters/opencode"
    "github.com/lukaszraczylo/harness-sync/internal/adapters/zed"
    "github.com/lukaszraczylo/harness-sync/internal/cli"
)

var version = "dev"

func main() {
    reg := adapter.NewRegistry()
    reg.Register(claudecode.New())
    reg.Register(crush.New())
    reg.Register(kilo.New())
    reg.Register(opencode.New())
    reg.Register(goose.New())
    reg.Register(cagent.New())
    reg.Register(zed.New())

    root := cli.NewRoot(version)
    root.AddCommand(cli.NewDetect(reg))

    if err := root.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

(If any `cagent.New()` / `zed.New()` etc. need a constructor signature different from this, update the adapter package — keep `New()` zero-arg.)

- [ ] **Step 2: Write the failing test**

`internal/cli/detect_test.go`:

```go
package cli

import (
    "bytes"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/canonical"
)

type detectableAdapter struct {
    name   string
    detect bool
}

func (d *detectableAdapter) Name() string                                  { return d.name }
func (d *detectableAdapter) Detect() bool                                  { return d.detect }
func (d *detectableAdapter) Render(_ *canonical.Bundle) (*adapter.FileSet, error) { return adapter.NewFileSet(), nil }
func (d *detectableAdapter) Import(_ string) (*adapter.ImportResult, error)       { return &adapter.ImportResult{}, nil }

func TestDetectCommandPrintsList(t *testing.T) {
    reg := adapter.NewRegistry()
    reg.Register(&detectableAdapter{name: "yes", detect: true})
    reg.Register(&detectableAdapter{name: "no", detect: false})

    var buf bytes.Buffer
    cmd := NewDetect(reg)
    cmd.SetOut(&buf)
    require.NoError(t, cmd.Execute())

    out := buf.String()
    assert.Contains(t, out, "yes")
    assert.Contains(t, out, "no")
    assert.Contains(t, out, "detected")
    assert.Contains(t, out, "not detected")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/cli/...`
Expected: FAIL with `undefined: NewDetect`

- [ ] **Step 4: Implement detect command**

`internal/cli/detect.go`:

```go
package cli

import (
    "fmt"

    "github.com/spf13/cobra"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func NewDetect(reg *adapter.Registry) *cobra.Command {
    return &cobra.Command{
        Use:   "detect",
        Short: "List adapters and whether each harness is present on this machine",
        RunE: func(cmd *cobra.Command, _ []string) error {
            for _, a := range reg.All() {
                status := "not detected"
                if a.Detect() {
                    status = "detected"
                }
                fmt.Fprintf(cmd.OutOrStdout(), "%-14s %s\n", a.Name(), status)
            }
            return nil
        },
    }
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/cli/... && go build ./... && ./harness-sync detect`
Expected: PASS; binary lists adapters with detection status

- [ ] **Step 6: Commit**

```bash
git add cmd internal/cli
git commit -m "feat(cli): wire adapters into registry and add detect subcommand"
```

---

### Task 20: apply command (single + all harnesses)

**Files:**
- Create: `internal/cli/apply.go`
- Create: `internal/cli/apply_test.go`

- [ ] **Step 1: Write the failing test**

`internal/cli/apply_test.go`:

```go
package cli

import (
    "bytes"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func TestApplyDryRun(t *testing.T) {
    root := t.TempDir()
    require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o644))
    require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o755))
    require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"),
        []byte("name: home\ngateway:\n  url: u\n  token: t\n  default_model: m\nmodels:\n  - {id: m}\n"), 0o644))

    reg := adapter.NewRegistry()
    reg.Register(&detectableAdapter{name: "stub", detect: true})

    var buf bytes.Buffer
    cmd := NewApply(reg)
    cmd.SetOut(&buf)
    cmd.SetArgs([]string{"--dry-run", "--root", root})
    require.NoError(t, cmd.Execute())

    assert.Contains(t, buf.String(), "dry-run")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/...`
Expected: FAIL with `undefined: NewApply`

- [ ] **Step 3: Implement apply command**

`internal/cli/apply.go`:

```go
package cli

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/apply"
    "github.com/lukaszraczylo/harness-sync/internal/canonical"
    "github.com/lukaszraczylo/harness-sync/internal/gitx"
)

func NewApply(reg *adapter.Registry) *cobra.Command {
    var (
        dryRun bool
        force  bool
        root   string
    )
    cmd := &cobra.Command{
        Use:   "apply [harness...]",
        Short: "Render canonical and propagate to harnesses",
        RunE: func(cmd *cobra.Command, args []string) error {
            if root == "" {
                h, err := os.UserHomeDir()
                if err != nil {
                    return err
                }
                root = filepath.Join(h, ".config", "harness-sync")
            }
            b, err := canonical.Load(root)
            if err != nil {
                return err
            }

            var selected []adapter.Adapter
            if len(args) == 0 {
                for _, a := range reg.All() {
                    if a.Detect() {
                        selected = append(selected, a)
                    }
                }
            } else {
                for _, name := range args {
                    a, ok := reg.Get(name)
                    if !ok {
                        return fmt.Errorf("unknown adapter %q", name)
                    }
                    selected = append(selected, a)
                }
            }

            opt := apply.Options{
                Bundle:   b,
                Adapters: selected,
                Repo:     gitx.New(root),
                DryRun:   dryRun,
                Force:    force,
            }
            rep, err := apply.Run(opt)
            if err != nil {
                return err
            }
            mode := "applied"
            if dryRun {
                mode = "dry-run"
            }
            fmt.Fprintf(cmd.OutOrStdout(), "%s: %d written, %d skipped, %d conflicts\n",
                mode, rep.Written, rep.Skipped, rep.Conflicts)
            for _, a := range rep.Actions {
                fmt.Fprintf(cmd.OutOrStdout(), "  %-10s %-12s %s %s\n", a.Adapter, a.Kind, a.Dest, a.Note)
            }
            if rep.Conflicts > 0 {
                return fmt.Errorf("%d conflicts; resolve .rej files", rep.Conflicts)
            }
            return nil
        },
    }
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print actions without writing")
    cmd.Flags().BoolVar(&force, "force", false, "overwrite without 3-way merge")
    cmd.Flags().StringVar(&root, "root", "", "canonical root (default ~/.config/harness-sync)")
    return cmd
}
```

- [ ] **Step 4: Wire into main**

Modify `cmd/harness-sync/main.go` to add `root.AddCommand(cli.NewApply(reg))` after `NewDetect`.

- [ ] **Step 5: Run tests + build**

Run: `go test ./... && go build ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli cmd
git commit -m "feat(cli): apply subcommand with dry-run, force, conflict reporting"
```

---

### Task 21: diff command

**Files:**
- Create: `internal/cli/diff.go`
- Create: `internal/cli/diff_test.go`

- [ ] **Step 1: Write the failing test**

`internal/cli/diff_test.go`:

```go
package cli

import (
    "bytes"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func TestDiffPrintsActions(t *testing.T) {
    root := t.TempDir()
    require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o644))
    require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o755))
    require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"),
        []byte("name: home\ngateway:\n  url: u\n  token: t\n  default_model: m\nmodels:\n  - {id: m}\n"), 0o644))

    reg := adapter.NewRegistry()
    reg.Register(&detectableAdapter{name: "stub", detect: true})

    var buf bytes.Buffer
    cmd := NewDiff(reg)
    cmd.SetOut(&buf)
    cmd.SetArgs([]string{"--root", root})
    require.NoError(t, cmd.Execute())
    assert.Contains(t, buf.String(), "dry-run")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/...`
Expected: FAIL with `undefined: NewDiff`

- [ ] **Step 3: Implement diff command**

`internal/cli/diff.go`:

```go
package cli

import (
    "github.com/spf13/cobra"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func NewDiff(reg *adapter.Registry) *cobra.Command {
    apply := NewApply(reg)
    cmd := &cobra.Command{
        Use:   "diff [harness...]",
        Short: "Show pending changes (alias for `apply --dry-run`)",
        RunE: func(cmd *cobra.Command, args []string) error {
            apply.SetArgs(append([]string{"--dry-run"}, args...))
            apply.SetOut(cmd.OutOrStdout())
            apply.SetErr(cmd.ErrOrStderr())
            return apply.Execute()
        },
    }
    return cmd
}
```

- [ ] **Step 4: Wire into main + run tests**

Add `root.AddCommand(cli.NewDiff(reg))`.

Run: `go test ./internal/cli/... && go build ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli cmd
git commit -m "feat(cli): diff subcommand as apply --dry-run shorthand"
```

---

### Task 22: profile command

**Files:**
- Create: `internal/cli/profile.go`
- Create: `internal/cli/profile_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/cli/profile_test.go`:

```go
package cli

import (
    "bytes"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestProfileList(t *testing.T) {
    root := t.TempDir()
    require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o755))
    require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "home.yaml"), []byte("name: home\n"), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "work.yaml"), []byte("name: work\n"), 0o644))

    var buf bytes.Buffer
    cmd := NewProfile()
    cmd.SetArgs([]string{"list", "--root", root})
    cmd.SetOut(&buf)
    require.NoError(t, cmd.Execute())

    out := buf.String()
    assert.Contains(t, out, "home")
    assert.Contains(t, out, "work")
}

func TestProfileUse(t *testing.T) {
    root := t.TempDir()
    require.NoError(t, os.MkdirAll(filepath.Join(root, "profiles"), 0o755))
    require.NoError(t, os.WriteFile(filepath.Join(root, "profiles", "work.yaml"), []byte("name: work\n"), 0o644))
    require.NoError(t, os.WriteFile(filepath.Join(root, "config.yaml"), []byte("active_profile: home\n"), 0o644))

    cmd := NewProfile()
    cmd.SetArgs([]string{"use", "work", "--root", root})
    require.NoError(t, cmd.Execute())

    body, err := os.ReadFile(filepath.Join(root, "config.yaml"))
    require.NoError(t, err)
    assert.Contains(t, string(body), "active_profile: work")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/...`
Expected: FAIL with `undefined: NewProfile`

- [ ] **Step 3: Implement profile command**

`internal/cli/profile.go`:

```go
package cli

import (
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"

    "github.com/spf13/cobra"
)

func NewProfile() *cobra.Command {
    var root string
    rootFlag := func(c *cobra.Command) {
        c.Flags().StringVar(&root, "root", "", "canonical root (default ~/.config/harness-sync)")
    }

    list := &cobra.Command{
        Use:   "list",
        Short: "List available profiles",
        RunE: func(cmd *cobra.Command, _ []string) error {
            r, err := resolveRoot(root)
            if err != nil {
                return err
            }
            entries, err := os.ReadDir(filepath.Join(r, "profiles"))
            if err != nil {
                return err
            }
            names := make([]string, 0, len(entries))
            for _, e := range entries {
                if strings.HasSuffix(e.Name(), ".yaml") {
                    names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
                }
            }
            sort.Strings(names)
            for _, n := range names {
                fmt.Fprintln(cmd.OutOrStdout(), n)
            }
            return nil
        },
    }
    rootFlag(list)

    use := &cobra.Command{
        Use:   "use <name>",
        Short: "Set active profile (rewrites config.yaml)",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            r, err := resolveRoot(root)
            if err != nil {
                return err
            }
            target := args[0]
            profPath := filepath.Join(r, "profiles", target+".yaml")
            if _, err := os.Stat(profPath); err != nil {
                return fmt.Errorf("profile %q not found at %s", target, profPath)
            }
            configPath := filepath.Join(r, "config.yaml")
            existing, _ := os.ReadFile(configPath)
            updated := setActiveProfile(string(existing), target)
            return os.WriteFile(configPath, []byte(updated), 0o644)
        },
    }
    rootFlag(use)

    cmd := &cobra.Command{Use: "profile", Short: "Manage profiles"}
    cmd.AddCommand(list, use)
    return cmd
}

func resolveRoot(flag string) (string, error) {
    if flag != "" {
        return flag, nil
    }
    h, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(h, ".config", "harness-sync"), nil
}

func setActiveProfile(existing, name string) string {
    lines := strings.Split(existing, "\n")
    for i, l := range lines {
        if strings.HasPrefix(l, "active_profile:") {
            lines[i] = "active_profile: " + name
            return strings.Join(lines, "\n")
        }
    }
    return strings.TrimRight(existing, "\n") + "\nactive_profile: " + name + "\n"
}
```

- [ ] **Step 4: Wire into main + run tests**

Add `root.AddCommand(cli.NewProfile())`.

Run: `go test ./internal/cli/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli cmd
git commit -m "feat(cli): profile list and profile use subcommands"
```

---

### Task 23: rollback + adapter list commands

**Files:**
- Create: `internal/cli/rollback.go`
- Create: `internal/cli/adapter.go`

- [ ] **Step 1: Implement rollback command**

`internal/cli/rollback.go`:

```go
package cli

import (
    "strconv"

    "github.com/spf13/cobra"

    "github.com/lukaszraczylo/harness-sync/internal/gitx"
)

func NewRollback() *cobra.Command {
    var root string
    cmd := &cobra.Command{
        Use:   "rollback [n]",
        Short: "Revert the last N apply commits in the canonical repo",
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            r, err := resolveRoot(root)
            if err != nil {
                return err
            }
            n := 1
            if len(args) == 1 {
                v, err := strconv.Atoi(args[0])
                if err != nil || v < 1 {
                    return cmd.Help()
                }
                n = v
            }
            return gitx.New(r).Revert(n)
        },
    }
    cmd.Flags().StringVar(&root, "root", "", "canonical root")
    return cmd
}
```

- [ ] **Step 2: Implement adapter list command**

`internal/cli/adapter.go`:

```go
package cli

import (
    "fmt"

    "github.com/spf13/cobra"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func NewAdapter(reg *adapter.Registry) *cobra.Command {
    list := &cobra.Command{
        Use:   "list",
        Short: "List registered adapters",
        RunE: func(cmd *cobra.Command, _ []string) error {
            for _, a := range reg.All() {
                fmt.Fprintln(cmd.OutOrStdout(), a.Name())
            }
            return nil
        },
    }
    cmd := &cobra.Command{Use: "adapter", Short: "Adapter introspection"}
    cmd.AddCommand(list)
    return cmd
}
```

- [ ] **Step 3: Wire both into main**

Add to `cmd/harness-sync/main.go`:

```go
root.AddCommand(cli.NewRollback())
root.AddCommand(cli.NewAdapter(reg))
```

- [ ] **Step 4: Run tests + build**

Run: `go test ./... && go build ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli cmd
git commit -m "feat(cli): rollback and adapter list subcommands"
```

---

### Task 24: init command (import flow)

**Files:**
- Create: `internal/cli/init.go`
- Create: `internal/cli/init_test.go`
- Create: `internal/ui/prompts.go`
- Create: `internal/ui/prompts_test.go`

- [ ] **Step 1: Write the failing UI tests**

`internal/ui/prompts_test.go`:

```go
package ui

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestMultiSelectNonInteractive(t *testing.T) {
    sel, err := MultiSelect("pick", []string{"a", "b", "c"}, WithNonInteractive([]string{"a", "c"}))
    assert.NoError(t, err)
    assert.Equal(t, []string{"a", "c"}, sel)
}

func TestMultiSelectNonInteractiveInvalidChoice(t *testing.T) {
    _, err := MultiSelect("pick", []string{"a", "b"}, WithNonInteractive([]string{"z"}))
    assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/...`
Expected: FAIL with `undefined: MultiSelect`

- [ ] **Step 3: Implement prompts with a non-interactive override for tests**

`internal/ui/prompts.go`:

```go
package ui

import (
    "fmt"

    "github.com/charmbracelet/huh"
)

type msOpts struct {
    nonInteractive []string
    interactive    bool
}

type Option func(*msOpts)

func WithNonInteractive(choices []string) Option {
    return func(o *msOpts) {
        o.nonInteractive = choices
        o.interactive = false
    }
}

func MultiSelect(title string, choices []string, opts ...Option) ([]string, error) {
    o := &msOpts{interactive: true}
    for _, fn := range opts {
        fn(o)
    }
    if !o.interactive {
        valid := map[string]bool{}
        for _, c := range choices {
            valid[c] = true
        }
        for _, c := range o.nonInteractive {
            if !valid[c] {
                return nil, fmt.Errorf("choice %q not in %v", c, choices)
            }
        }
        return o.nonInteractive, nil
    }
    var picked []string
    opts2 := make([]huh.Option[string], 0, len(choices))
    for _, c := range choices {
        opts2 = append(opts2, huh.NewOption(c, c))
    }
    form := huh.NewForm(huh.NewGroup(
        huh.NewMultiSelect[string]().
            Title(title).
            Options(opts2...).
            Value(&picked),
    ))
    if err := form.Run(); err != nil {
        return nil, err
    }
    return picked, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/...`
Expected: PASS

- [ ] **Step 5: Write failing init test**

`internal/cli/init_test.go`:

```go
package cli

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/canonical"
)

type importableAdapter struct {
    name string
    res  *adapter.ImportResult
}

func (i *importableAdapter) Name() string                                  { return i.name }
func (i *importableAdapter) Detect() bool                                  { return true }
func (i *importableAdapter) Render(_ *canonical.Bundle) (*adapter.FileSet, error) { return adapter.NewFileSet(), nil }
func (i *importableAdapter) Import(_ string) (*adapter.ImportResult, error)        { return i.res, nil }

func TestInitImportWritesCanonical(t *testing.T) {
    root := t.TempDir()
    home := t.TempDir()

    reg := adapter.NewRegistry()
    reg.Register(&importableAdapter{
        name: "stub",
        res: &adapter.ImportResult{
            Skills:       []canonical.Skill{{Name: "hi", Body: "---\nname: hi\n---\nhi body", Path: "hi/SKILL.md"}},
            Instructions: "# global",
            MCP:          []canonical.MCPServer{{Name: "filepuff", Command: "/bin/x"}},
        },
    })

    cmd := NewInit(reg)
    cmd.SetArgs([]string{"--root", root, "--home", home, "--from", "stub", "--no-prompt"})
    require.NoError(t, cmd.Execute())

    assert.FileExists(t, filepath.Join(root, "config.yaml"))
    assert.FileExists(t, filepath.Join(root, "instructions", "global.md"))
    assert.FileExists(t, filepath.Join(root, "mcp.yaml"))
    assert.FileExists(t, filepath.Join(root, "skills", "hi", "SKILL.md"))
    _, err := os.Stat(filepath.Join(root, ".git"))
    require.NoError(t, err)
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/cli/...`
Expected: FAIL with `undefined: NewInit`

- [ ] **Step 7: Implement init command**

`internal/cli/init.go`:

```go
package cli

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"

    "github.com/lukaszraczylo/harness-sync/internal/adapter"
    "github.com/lukaszraczylo/harness-sync/internal/canonical"
    "github.com/lukaszraczylo/harness-sync/internal/gitx"
    "github.com/lukaszraczylo/harness-sync/internal/render"
    "github.com/lukaszraczylo/harness-sync/internal/ui"
)

func NewInit(reg *adapter.Registry) *cobra.Command {
    var (
        root      string
        home      string
        from      []string
        noPrompt  bool
    )
    cmd := &cobra.Command{
        Use:   "init",
        Short: "Initialise canonical config from existing harnesses",
        RunE: func(cmd *cobra.Command, _ []string) error {
            r, err := resolveRoot(root)
            if err != nil {
                return err
            }
            if home == "" {
                home, err = os.UserHomeDir()
                if err != nil {
                    return err
                }
            }
            if err := os.MkdirAll(r, 0o755); err != nil {
                return err
            }
            candidates := reg.DetectedNames()
            if len(candidates) == 0 {
                return fmt.Errorf("no harnesses detected under %s", home)
            }
            var picked []string
            if len(from) > 0 {
                picked = from
            } else if noPrompt {
                picked = candidates
            } else {
                picked, err = ui.MultiSelect("Import from which harnesses?", candidates)
                if err != nil {
                    return err
                }
            }
            return runImport(cmd, reg, r, home, picked)
        },
    }
    cmd.Flags().StringVar(&root, "root", "", "canonical root (default ~/.config/harness-sync)")
    cmd.Flags().StringVar(&home, "home", "", "home dir (default $HOME)")
    cmd.Flags().StringSliceVar(&from, "from", nil, "import from specific adapter(s), skip prompt")
    cmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "import from all detected adapters without prompting")
    return cmd
}

func runImport(cmd *cobra.Command, reg *adapter.Registry, root, home string, picked []string) error {
    bundle := &canonical.Bundle{
        Root:   root,
        Config: canonical.Config{ActiveProfile: "imported"},
    }
    seenSkills := map[string]bool{}
    seenAgents := map[string]bool{}
    seenMCP := map[string]bool{}
    var instructions []string

    for _, name := range picked {
        ad, ok := reg.Get(name)
        if !ok {
            return fmt.Errorf("unknown adapter %q", name)
        }
        res, err := ad.Import(home)
        if err != nil {
            return fmt.Errorf("import %s: %w", name, err)
        }
        for _, s := range res.Skills {
            key := s.Name + "|" + s.Body
            if seenSkills[key] {
                continue
            }
            seenSkills[key] = true
            bundle.Skills = append(bundle.Skills, s)
        }
        for _, a := range res.Agents {
            key := a.Name + "|" + a.Body
            if seenAgents[key] {
                continue
            }
            seenAgents[key] = true
            bundle.Agents = append(bundle.Agents, a)
        }
        for _, m := range res.MCP {
            if seenMCP[m.Name] {
                continue
            }
            seenMCP[m.Name] = true
            bundle.MCP.Servers = append(bundle.MCP.Servers, m)
        }
        if res.Instructions != "" {
            instructions = append(instructions, fmt.Sprintf("<!-- from %s -->\n%s", name, res.Instructions))
        }
    }
    bundle.Instructions.Global = strings.Join(instructions, "\n\n")

    if err := writeCanonical(root, bundle, picked); err != nil {
        return err
    }
    repo := gitx.New(root)
    if !repo.IsRepo() {
        if err := repo.Init(); err != nil {
            return err
        }
        if err := repo.Configure("harness-sync", "harness-sync@localhost"); err != nil {
            return err
        }
    }
    if err := repo.AddAll(); err != nil {
        return err
    }
    if err := repo.Commit(fmt.Sprintf("import from %s", strings.Join(picked, ", "))); err != nil {
        return err
    }
    fmt.Fprintf(cmd.OutOrStdout(), "imported from %v into %s\n", picked, root)
    return nil
}

func writeCanonical(root string, b *canonical.Bundle, picked []string) error {
    if err := os.MkdirAll(filepath.Join(root, "profiles"), 0o755); err != nil {
        return err
    }
    if err := os.MkdirAll(filepath.Join(root, "skills"), 0o755); err != nil {
        return err
    }
    if err := os.MkdirAll(filepath.Join(root, "agents"), 0o755); err != nil {
        return err
    }
    if err := os.MkdirAll(filepath.Join(root, "instructions"), 0o755); err != nil {
        return err
    }

    cfgBody, err := render.YAML(map[string]any{
        "enabled_harnesses": picked,
        "active_profile":    "imported",
    })
    if err != nil {
        return err
    }
    if err := os.WriteFile(filepath.Join(root, "config.yaml"), cfgBody, 0o644); err != nil {
        return err
    }

    profBody, err := render.YAML(canonical.Profile{
        Name: "imported",
        Gateway: canonical.Gateway{
            URL:          "https://gateway.local",
            Token:        "dummy",
            DefaultModel: "claude-sonnet-4-6",
        },
        Models: []canonical.Model{{ID: "claude-sonnet-4-6", Alias: "sonnet"}},
    })
    if err != nil {
        return err
    }
    if err := os.WriteFile(filepath.Join(root, "profiles", "imported.yaml"), profBody, 0o644); err != nil {
        return err
    }

    if len(b.MCP.Servers) > 0 {
        mcpBody, err := render.YAML(b.MCP)
        if err != nil {
            return err
        }
        if err := os.WriteFile(filepath.Join(root, "mcp.yaml"), mcpBody, 0o644); err != nil {
            return err
        }
    }

    for _, s := range b.Skills {
        path := filepath.Join(root, "skills", s.Name, "SKILL.md")
        if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
            return err
        }
        if err := os.WriteFile(path, []byte(s.Body), 0o644); err != nil {
            return err
        }
    }
    for _, a := range b.Agents {
        path := filepath.Join(root, "agents", a.Name+".md")
        if err := os.WriteFile(path, []byte(a.Body), 0o644); err != nil {
            return err
        }
    }
    if b.Instructions.Global != "" {
        if err := os.WriteFile(filepath.Join(root, "instructions", "global.md"),
            []byte(b.Instructions.Global), 0o644); err != nil {
            return err
        }
    }
    return nil
}
```

- [ ] **Step 8: Wire into main + run all tests**

Add `root.AddCommand(cli.NewInit(reg))`.

Run: `go test ./... && go build ./...`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/cli internal/ui cmd
git commit -m "feat(cli): init subcommand imports detected harnesses with dedupe + git commit"
```

---

## Phase 7: Integration test + polish

### Task 25: End-to-end integration test

**Files:**
- Create: `tests/e2e/e2e_test.go`

- [ ] **Step 1: Write the integration test**

`tests/e2e/e2e_test.go`:

```go
package e2e

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestInitApplyDiffCycle(t *testing.T) {
    bin := buildBinary(t)
    home := t.TempDir()
    root := filepath.Join(home, ".config", "harness-sync")

    // Fake a claude-code install
    require.NoError(t, os.MkdirAll(filepath.Join(home, ".claude", "skills", "hi"), 0o755))
    require.NoError(t, os.WriteFile(
        filepath.Join(home, ".claude", "skills", "hi", "SKILL.md"),
        []byte("---\nname: hi\n---\nbody"), 0o644))

    run := func(args ...string) ([]byte, error) {
        cmd := exec.Command(bin, args...)
        cmd.Env = append(os.Environ(), "HOME="+home)
        return cmd.CombinedOutput()
    }

    out, err := run("init", "--no-prompt")
    require.NoError(t, err, string(out))
    assert.FileExists(t, filepath.Join(root, "config.yaml"))
    assert.FileExists(t, filepath.Join(root, "skills", "hi", "SKILL.md"))

    out, err = run("apply")
    require.NoError(t, err, string(out))

    out, err = run("diff")
    require.NoError(t, err, string(out))
    assert.NotContains(t, string(out), "conflict")
}

func buildBinary(t *testing.T) string {
    t.Helper()
    bin := filepath.Join(t.TempDir(), "harness-sync")
    cmd := exec.Command("go", "build", "-o", bin, "./cmd/harness-sync")
    cmd.Dir = "../.."
    out, err := cmd.CombinedOutput()
    require.NoError(t, err, string(out))
    return bin
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./tests/e2e/... -count=1`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add tests/e2e
git commit -m "test(e2e): init-apply-diff cycle through real binary"
```

---

### Task 26: README and final polish

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Write user-facing README**

Replace `README.md` with:

```markdown
# harness-sync

Sync skills, agents, MCP, and LLM endpoints across multiple LLM harnesses.

Canonical config lives in `~/.config/harness-sync/`, tracked in git.
On `apply`, harness-sync renders the canonical config into each detected
harness's native layout (symlinks where shapes match, transformed files
where they differ) and uses `git merge-file` to resolve conflicts.

## Supported harnesses

- claude-code   (`~/.claude/`)
- crush         (`~/.config/crush/`)
- kilo          (`~/.config/kilo/`)
- opencode      (`~/.config/opencode/`)
- goose         (`~/.config/goose/`)
- cagent        (`~/.config/cagent/`)
- zed           (`~/.config/zed/`)

Add new harnesses by writing a Go package under `internal/adapters/`
and registering it in `cmd/harness-sync/main.go`.

## Install

    make build && make install

## First run

    harness-sync detect          # see what's installed
    harness-sync init            # pick which to import from; canonical tree created
    harness-sync apply           # propagate to all detected harnesses

## Day-to-day

    harness-sync diff            # show pending changes
    harness-sync apply           # apply them
    harness-sync profile use work
    harness-sync rollback 1      # revert last apply

## Conflict resolution

If you've edited a file harness-sync manages and `apply` cannot merge
cleanly, the merged-with-markers result is written next to the target
as `<file>.rej`. Resolve, copy back, run `apply` again.

## Design

See [`docs/superpowers/specs/2026-05-23-harness-sync-design.md`](docs/superpowers/specs/2026-05-23-harness-sync-design.md).
```

- [ ] **Step 2: Final build and test**

Run: `go test ./... && go build ./... && ./harness-sync --help`
Expected: PASS; help output lists init/detect/diff/apply/rollback/profile/adapter

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: user-facing README with install, first-run, and conflict guidance"
```

---

## Self-review checklist (for the implementer at the end)

- [ ] Every spec section maps to at least one task: canonical layout (Tasks 2–3), profile model (Task 3 fixtures + Task 24), adapter interface (Tasks 5–6), sync mechanics per asset (Tasks 11–18), commands (Tasks 19–24), conflict handling (Task 10), import flow (Task 24), error handling (apply pipeline in Task 10), testing strategy (per-task tests + Task 25), module layout (matches Tasks 1–26).
- [ ] No `TBD`, no `add appropriate error handling` placeholders. All adapter-specific schema steps include the inspection command and constrain the engineer to "read the actual config files; match the keys."
- [ ] Type names consistent across tasks: `Bundle`, `Profile`, `MCPRegistry`, `MCPServer`, `Adapter`, `FileSet`, `File`, `Kind`, `Registry`, `ImportResult`, `apply.Options`, `apply.Report`, `apply.Action`, `gitx.Repo`, `merge.Inputs`, `merge.Result`.
- [ ] Each task includes failing test → implementation → passing test → commit.

---

## Acceptance criteria (mapped from spec)

1. `harness-sync init` on a fresh machine populates `~/.config/harness-sync/` from selected harnesses (Task 24, Task 25).
2. `harness-sync apply` renders correct native files; subsequent `harness-sync diff` reports zero changes (Tasks 20, 21, 25).
3. After hand-editing a synced file, the next `apply` 3-way-merges or writes `.rej` (Task 10 conflict test + Task 25).
4. `harness-sync profile use work` switches profile (Task 22); next `apply` re-renders LLM configs.
5. Adding a new harness: new package + one registry line in `cmd/harness-sync/main.go` (Tasks 13–18 demonstrate the pattern).
6. Every adapter has golden-file tests; integration test in Task 25 covers init → apply → diff.
