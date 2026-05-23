# harness-sync

One canonical source of truth for skills, agents, MCP servers, LLM endpoints, and global instructions across multiple LLM harnesses. Edit once at `~/.config/harness-sync/`; propagate everywhere.

## Supported harnesses

| Harness | Config path | What harness-sync writes |
|---|---|---|
| claude-code | `~/.claude/` | `skills/` (symlink), `agents/` (symlink), `CLAUDE.md`, `settings.json` `mcpServers` (merged) |
| crush | `~/.config/crush/` | `skills/` (symlink), `crush.json` `providers`/`default_model`/`mcp` (merged) |
| kilo | `~/.config/kilo/` | `agent/` (symlink), `kilo.json` `model`/`mcp` (merged) |
| opencode | `~/.config/opencode/` | `opencode.jsonc` `provider`/`model`/`mcp` (merged), `AGENTS.md` |
| goose | `~/.config/goose/` | `config.yaml` `GOOSE_PROVIDER`/`GOOSE_MODEL`/`extensions` (merged) |
| cagent | `~/.config/cagent/` | starter `default.yaml` (per-run config) |
| zed | `~/.config/zed/` | `settings.json` `agent`/`context_servers` (merged — other keys preserved) |

Adding a new harness: drop a Go package under `internal/adapters/<name>/` implementing `adapter.Adapter`, register it in `cmd/harness-sync/main.go`.

## Install

```bash
make build && make install   # installs to ~/.local/bin/harness-sync
```

## First run

```bash
harness-sync detect          # see which harnesses are installed
harness-sync init            # pick which to import from; canonical tree created at ~/.config/harness-sync/
harness-sync apply           # propagate canonical to all detected harnesses
```

`init` opens an interactive multi-select of detected harnesses. Use `--no-prompt` to import from all, or `--from claude-code,crush` to be explicit.

## Day-to-day

```bash
harness-sync show              # list files each adapter would write
harness-sync diff              # apply --dry-run shorthand
harness-sync apply             # propagate canonical
harness-sync apply crush       # propagate to one harness only
harness-sync profile use work  # switch active profile and reapply
harness-sync rollback 1        # revert last apply via git revert
harness-sync adapter list      # print registered adapters
```

## Canonical layout

```
~/.config/harness-sync/
├── config.yaml                # enabled_harnesses, active_profile
├── profiles/
│   ├── home.yaml              # gateway URL + token + model allowlist + upstreams
│   └── work.yaml
├── instructions/
│   ├── global.md              # rendered to CLAUDE.md / AGENTS.md / etc.
│   └── per-harness/<name>.md  # optional override
├── skills/<name>/SKILL.md     # markdown + frontmatter; symlinked into harnesses that accept it
├── agents/<name>.md           # markdown + frontmatter
├── mcp.yaml                   # canonical MCP server registry
├── state/<harness>/...        # last-rendered snapshots (git-tracked, do not hand-edit)
└── backups/                   # pre-apply backups; git-ignored
```

The whole tree is a git repository. Push it to a private remote for cross-machine sync.

## Profiles

A profile bundles the LLM stack. Switching profiles re-renders every harness's LLM config in one command.

```yaml
# profiles/home.yaml
name: home
gateway:
  url: https://gateway.lan
  token: dummy-local-token
  default_model: claude-sonnet-4-6
upstreams:
  - name: anthropic
    api_key: ${ANTHROPIC_API_KEY}
  - name: openai
    api_key: ${OPENAI_API_KEY}
models:
  - id: claude-sonnet-4-6
    alias: sonnet
  - id: claude-opus-4-7
    alias: opus
```

Secrets use `${VAR}` for env-var substitution at render time. The dummy gateway token may be plaintext (it has no value if leaked); real provider keys must be env-var references.

## Conflict resolution

`apply` performs git-style three-way merge on every rendered file. The base is the previous render snapshot from `state/<harness>/`; "ours" is the new render; "theirs" is the current target file. When the merge is clean, the target is updated. When it conflicts, harness-sync writes the merged-with-markers result next to the target as `<file>.rej`, leaves the original file untouched, and exits non-zero. Resolve the conflict (edit the target, delete `.rej`), then run `apply` again.

The same canonical repo holds the state snapshots. `rollback N` calls `git revert` on the last N commits and reapplies.

## Design and implementation

- Design: [`docs/superpowers/specs/2026-05-23-harness-sync-design.md`](docs/superpowers/specs/2026-05-23-harness-sync-design.md)
- Plan: [`docs/superpowers/plans/2026-05-23-harness-sync.md`](docs/superpowers/plans/2026-05-23-harness-sync.md)

## License

Same as the surrounding repo.
