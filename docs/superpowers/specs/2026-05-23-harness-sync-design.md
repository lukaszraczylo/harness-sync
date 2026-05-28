# harness-sync — design

**Status:** draft (pending user approval)
**Date:** 2026-05-23
**Author:** Lukasz Raczylo

## Problem

Lukasz uses multiple LLM coding harnesses (claude code, crush, kilo, opencode, goose, cagent, zed, others). Each maintains its own copy of:

- **Skills** (Claude-style markdown + frontmatter)
- **Agents / subagents** (markdown with tool allowlists)
- **LLM endpoint and provider configuration** (gateway URL + token + model list)
- **MCP server registry**
- **Global instructions** (`CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, etc.)

The duplication causes drift, manual copy-paste maintenance, and inconsistent capabilities across harnesses. Lukasz also runs a personal LLM gateway with a dummy token that fronts multiple upstream providers (anthropic, openai, openrouter, groq, ollama, etc.); harnesses should point at the gateway, not at providers directly.

## Goal

A single Go binary, `harness-sync`, that:

1. Maintains `~/.config/harness-sync/` as the canonical source of truth for all of the above.
2. On `apply`, propagates canonical content into each detected harness, using symlinks where formats match and rendered/transformed files where they differ.
3. Uses git to track render history and resolve conflicts.
4. Supports a plugin-style adapter contract so new harnesses can be added by dropping in a Go package and one registry line.

Non-goals (v1): watcher daemon, GUI, remote sync (user has git), keychain/Vault backends beyond env-var substitution.

## Architecture

Single binary, subcommand CLI. Three layers:

- **Canonical** — hand-edited YAML + markdown under `~/.config/harness-sync/`.
- **Adapters** — Go interface implementations, one Go package per supported harness under `internal/adapters/<name>/`. Each adapter knows its harness's file paths and how to render canonical → native schema.
- **State** — last-rendered snapshot tracked in the same git repo, used as merge base for conflict detection.

Flow per `apply`:

```
canonical (YAML/MD) ──► adapter.Render() ──► rendered_new ──┐
                                                            ├─► 3-way merge ──► target file
git HEAD: state/<harness>/<path> ──► rendered_old ──────────┤
target filesystem ──► target_current ───────────────────────┘
                          │
                          └─► after write: stage state/, git commit
```

## Canonical layout

```
~/.config/harness-sync/
├── config.yaml                  # global: enabled harnesses, default profile, paths
├── profiles/
│   ├── home.yaml                # gateway URL, dummy token, model allowlist, upstream providers
│   ├── work.yaml
│   └── offline.yaml             # e.g. local ollama only
├── instructions/
│   ├── global.md                # rendered to CLAUDE.md / AGENTS.md / GEMINI.md / opencode AGENTS.md per harness
│   └── per-harness/<name>.md    # optional override
├── skills/                      # markdown + frontmatter; claude-code shape is canonical
│   └── <skill-name>/SKILL.md
├── agents/                      # markdown + frontmatter
│   └── <agent-name>.md
├── mcp.yaml                     # canonical MCP server registry
├── state/                       # git-tracked render snapshots — DO NOT EDIT BY HAND
│   └── <harness>/...
└── backups/                     # pre-apply backups; git-ignored
```

`~/.config/harness-sync/` is itself a git repository. The user may push it to a private remote for cross-machine sync.

## Profile model

A profile bundles the LLM stack. Harnesses bind to the active profile (named in `config.yaml`). Switching profiles re-renders all harness LLM configs in one command.

```yaml
# profiles/home.yaml
name: home
gateway:
  url: https://gateway.lan.example.com
  token: dummy-local-token            # plaintext OK; gateway accepts any non-empty token
  default_model: claude-sonnet-4-6
upstreams:                            # informational: gateway already knows these; harnesses see only models
  - name: anthropic
    api_key: ${ANTHROPIC_API_KEY}     # placeholder preserved; resolved by the harness, never plaintext
  - name: openai
    api_key: ${OPENAI_API_KEY}
  - name: openrouter
    api_key: ${OPENROUTER_API_KEY}
  - name: ollama
    base_url: http://10.0.1.21:11434
models:                               # allowlist exposed to harnesses
  - id: claude-sonnet-4-6
    alias: sonnet
  - id: claude-opus-4-7
    alias: opus
  - id: gpt-4o
    alias: gpt
```

Secret handling: dummy gateway token is plaintext (no value if leaked). Any real provider keys reference env vars via `${VAR}`. harness-sync does NOT resolve `${VAR}` — it writes the placeholder through to the rendered configs verbatim (translating to `{env:VAR}` for opencode/kilo), so neither the canonical tree nor the rendered files nor the git-tracked `state/` snapshots ever contain plaintext provider keys; each harness resolves the reference at use time.

## Adapter interface

```go
package adapter

type Adapter interface {
    Name() string                                    // "crush", "claude-code", ...
    Detect() bool                                    // true if harness config dir exists on this machine
    Targets() []Target                               // declarative list of what gets written where
    Render(c *Canonical, p *Profile) (FileSet, error)
    Import(fs Filesystem) (CanonicalDelta, error)    // reverse: read native files, produce canonical deltas
}

type Target struct {
    Kind       Kind     // SymlinkDir | SymlinkFile | RenderedFile
    SourcePath string   // path within canonical (e.g. "skills/")
    DestPath   string   // absolute target path (expanded; e.g. "/home/nvm/.config/crush/prompts")
    Format     Format   // JSON | TOML | YAML | Markdown | Raw
}

type FileSet map[string]File   // dest path -> rendered file (bytes + perms + symlink target)
```

Each adapter is expected to be roughly 150–300 LOC: declare targets, marshal canonical structs into native schema, handle import.

Adapters live at `internal/adapters/<name>/` and self-register through a static slice. No process-boundary plugin system — YAGNI.

### v1 adapter set

- `claudecode` — `~/.claude/{skills,agents,CLAUDE.md}` + `~/.claude/settings.json` (MCP block)
- `crush` — `~/.config/crush/`
- `kilo` — `~/.config/kilo/`
- `opencode` — `~/.config/opencode/`
- `goose` — `~/.config/goose/`
- `cagent` — `~/.config/cagent/`
- `zed` — `~/.config/zed/`

Each ships with golden-file tests under `internal/adapters/<name>/testdata/`.

## Sync mechanics by asset

| Asset | Mechanism | Reasoning |
|---|---|---|
| `skills/` | symlink directory into harness's expected location (or render if shape differs) | bulk, frequently edited, identical shape across claude-code-likes |
| `agents/` | symlink directory or render to native list file | same |
| `profiles/<active>.yaml` → endpoint config | **render** to native schema (per-harness keys, format) | each harness uses different keys (`providers`, `models`, `endpoints`) and serialization |
| `mcp.yaml` → harness MCP block | render to native (JSON/YAML/TOML) | shape and embedding location vary |
| `instructions/global.md` | render/copy to harness's expected filename and path | identical content, different filename per harness |

## Commands

```
harness-sync init                    # first run: scan ~/.config + ~/.claude, detect harnesses,
                                     # offer multi-select import, merge+dedupe into canonical,
                                     # git init, initial commit
harness-sync detect                  # list detected harnesses and their config paths
harness-sync diff [harness]          # show pending changes (canonical vs target)
harness-sync apply [harness] [--dry-run] [--force]
                                     # render + write; default does 3-way merge,
                                     # --force overwrites without merge, --dry-run shows plan only
harness-sync rollback [n]            # git revert N commits in canonical repo, then reapply
harness-sync profile use <name>      # switch active profile in config.yaml + reapply
harness-sync profile list
harness-sync adapter list            # show registered adapters + detection status
```

Default behavior with no subcommand: print `detect` summary + suggest `diff` or `apply`.

## Conflict handling

`apply` performs git-style three-way merge per rendered file:

1. `rendered_new` = adapter output for this run.
2. `rendered_old` = `state/<harness>/<path>` at git HEAD (or empty if first apply).
3. `target_current` = current contents at destination (or empty if missing).

```
if target_current == rendered_old:
    fast-path write rendered_new           # clean, no manual edits since last apply
elif target_current == rendered_new:
    skip                                   # already in sync
else:
    git merge-file rendered_new rendered_old target_current
    on clean merge: write result
    on conflict:   write .rej file alongside target, skip, accumulate report
```

After all adapters run successfully, stage and commit the new `state/` tree:

```
git add state/ && git commit -m "apply: <harnesses> profile=<name>"
```

Rollback: `git revert` on the canonical repo, then `apply` again to push the reverted state outward.

Symlink conflicts: if target should be a symlink and is currently a regular file (or a symlink elsewhere), move it to `backups/<harness>/<timestamp>/` and replace with the correct symlink.

## Import flow (first run)

`harness-sync init` on empty `~/.config/harness-sync/`:

1. Scan `~/.config/*` and `~/.claude/` for known harness layouts (using each adapter's `Detect()`).
2. Print discovered harnesses with checkboxes:
   ```
   Found harnesses. Import from which?
   [x] claude-code  (~/.claude/)
   [x] crush        (~/.config/crush/)
   [x] kilo         (~/.config/kilo/)
   [ ] cagent       (~/.config/cagent/)
   ...
   ```
3. For each selected harness, invoke `adapter.Import()` which reads native files and yields a `CanonicalDelta`.
4. Merge deltas:
   - **skills/agents:** dedupe by filename + content hash. Identical → keep one. Divergent → write all variants as `<name>.from-<harness>.md` and emit a TODO for the user to resolve.
   - **profiles:** create one `imported.yaml` aggregating discovered endpoints; user edits later.
   - **mcp.yaml:** union by server name; conflicts on same name go into a `# CONFLICT:` block.
   - **instructions:** concatenate sourced files into `instructions/imported/<harness>.md`; user later promotes a unified version to `global.md`.
5. `git init && git add -A && git commit -m "import from {harnesses}"`.

After init the user has a fully populated, version-controlled canonical tree that they then prune and curate.

## Error handling

- Errors wrapped with adapter + operation context: `fmt.Errorf("adapter %s: render %s: %w", ...)`.
- `apply` is atomic per-file but not across the run: a fatal error in one adapter does not roll back files already written by previous adapters; the run report lists all writes and failures.
- On partial failure, the `state/` commit is **not** created — next run will re-attempt and detect divergence cleanly.
- Symlink writes go through `os.Symlink` + atomic rename via tempfile to avoid races.

## Testing strategy

- **Unit tests per adapter**: table-driven, with `testdata/` golden fixtures. Input: canonical struct + profile. Output: expected rendered files. Diffs surface on regenerate.
- **Integration test**: build the binary, run against a tempdir HOME populated with fake harness configs, assert `init → apply → diff` is empty.
- **Merge tests**: cover the three-way merge logic with synthetic `rendered_old`, `rendered_new`, `target_current` permutations.

## Logging and observability

- Default output: one line per harness with summary counts (`crush: 12 files synced, 0 conflicts`).
- `--verbose`: per-file actions.
- `--quiet`: errors only.
- No log file; structured logs to stderr if `HARNESS_SYNC_LOG_JSON=1`.

## Module layout

```
harness-sync/
├── cmd/harness-sync/main.go      # cobra/cli entrypoint
├── internal/
│   ├── canonical/                # canonical struct types + loaders
│   ├── adapter/                  # Adapter interface, registry, common helpers
│   ├── adapters/
│   │   ├── claudecode/
│   │   ├── crush/
│   │   ├── kilo/
│   │   ├── opencode/
│   │   ├── goose/
│   │   ├── cagent/
│   │   └── zed/
│   ├── merge/                    # 3-way merge wrapper around git merge-file
│   ├── git/                      # thin wrapper over `git` CLI invocation
│   ├── render/                   # format-specific marshallers (JSON/TOML/YAML)
│   ├── secrets/                  # env-var substitution
│   └── ui/                       # interactive prompts (init multi-select)
├── docs/superpowers/specs/2026-05-23-harness-sync-design.md
├── go.mod
└── README.md
```

## Open questions / decisions to confirm

- **CLI library:** `spf13/cobra` (familiar, batteries included) vs `urfave/cli` (lighter). Recommend cobra.
- **Interactive prompts:** `charmbracelet/huh` or `AlecAivazis/survey`. Recommend huh (aligns with crush's stack).
- **YAML library:** `goccy/go-yaml` (preserves comments) vs `gopkg.in/yaml.v3`. Recommend goccy since canonical YAML is hand-edited and comments matter.
- **Adapter discovery:** static registry slice (decided: yes; simple, fast).

## Grounded harness schemas

Verified from live config files and official docs (2026-05-23).

### Config file locations and top-level key ownership

| Harness | Config file | Managed by harness-sync | User-managed (preserve) |
|---|---|---|---|
| claude-code | `~/.claude/settings.json` | `mcpServers` | `hooks`, `permissions`, `env`, and everything else |
| crush | `~/.config/crush/crush.json` | `providers`, `default_model`, `mcp` | `$schema`, `lsp`, `options`, `permissions` |
| kilo | `~/.config/kilo/kilo.json` | `model`, `mcp` | `$schema`, `small_model`, `instructions`, `permission`, `compaction`, `watcher`, `formatter`, `skills` |
| opencode | `~/.config/opencode/opencode.jsonc` | `provider`, `model`, `mcp` | `$schema`, `agent`, `instructions`, and everything else |
| zed | `~/.config/zed/settings.json` | `agent`, `context_servers` | everything else (theme, fonts, keybindings, …) |
| goose | `~/.config/goose/config.yaml` | `GOOSE_PROVIDER`, `GOOSE_MODEL`, `extensions` | `GOOSE_TEMPERATURE` and everything else |

### MCP / server key names

| Harness | Key | Entry shape |
|---|---|---|
| claude-code | `mcpServers` (map) | `{command, args, env, url, transport}` — no `type` field |
| crush | `mcp` (map) | `{type: stdio\|http\|sse, command, args, env}` or `{type: http\|sse, url}` |
| kilo | `mcp` (map) | `{type: local\|remote, command: []string, enabled}` or `{type: remote, url, enabled}` |
| opencode | `mcp` (map) | `{type: local\|remote, command: []string, enabled}` or `{type: remote, url, enabled}` |
| zed | `context_servers` (map) | `{enabled, source: "custom", command, args}` or `{enabled, url}` |
| goose | `extensions` (map) | `{type: stdio, cmd, args, enabled}` — no MCP key; goose extension format |

### Model key names

| Harness | Provider key | Model key |
|---|---|---|
| claude-code | — (none) | `env.ANTHROPIC_DEFAULT_MODEL` (string) |
| crush | `providers` (array of objects) | `default_model` (string) |
| kilo | — (no providers) | `model` (string, e.g. `"llmgw/anthropic/claude-sonnet-4-6"`) |
| opencode | `provider` (singular, map keyed by name) | `model` (string) |
| zed | — | `agent.default_model` (object `{provider, model}`) |
| goose | `GOOSE_PROVIDER` (string) | `GOOSE_MODEL` (string) |

### Merge policy

All adapters write config files using **merge-not-replace**: existing top-level keys not in the managed set are preserved verbatim. This prevents harness-sync from destroying user configuration on every apply.

## Acceptance criteria

1. `harness-sync init` on a fresh machine with several harness configs produces a populated, git-committed canonical tree without data loss.
2. `harness-sync apply` renders correct native files into each detected harness; subsequent `harness-sync diff` is empty.
3. After hand-editing a synced file, the next `apply` detects the divergence and either three-way-merges cleanly or writes a `.rej` and exits non-zero.
4. `harness-sync profile use work` re-renders LLM endpoint config in every harness atomically.
5. Adding a new harness requires only a new package under `internal/adapters/` and one line in the registry; existing adapters untouched.
6. All adapters have golden-file tests; integration test covers the init → apply → diff cycle.
