# harness-sync

Sync skills, agents, MCP, and LLM endpoints across multiple LLM harnesses
(claude-code, crush, kilo, opencode, goose, cagent, zed).

Canonical config lives in `~/.config/harness-sync/` (git-tracked). Run
`harness-sync apply` to propagate.

See `docs/superpowers/specs/2026-05-23-harness-sync-design.md` for design.

## Build

    make build && make install
