// Package claudecode is the claude-code harness adapter.
package claudecode

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "claude-code"

// Adapter implements adapter.Adapter for the claude-code harness.
type Adapter struct {
	*adapter.Base
}

// WithHome overrides the home directory used to resolve target paths.
func WithHome(h string) adapter.BaseOption { return func(b *adapter.Base) { b.Home = h } }

// New returns a new Adapter with the given options applied.
func New(opts ...adapter.BaseOption) *Adapter {
	return &Adapter{Base: adapter.NewBase(opts...)}
}

// Name returns the harness identifier.
func (a *Adapter) Name() string { return name }

// Capabilities declares what claude-code harness-sync manages.
// Claude Code has its own Anthropic Max subscription — provider/model config is skipped.
func (a *Adapter) Capabilities() adapter.HarnessCapabilities {
	return adapter.HarnessCapabilities{
		ManagesProviders:    false,
		ManagesModels:       false,
		ManagesMCP:          true,
		ManagesSkills:       true,
		ManagesAgents:       true,
		ManagesRules:        true,
		ManagesInstructions: true,
		HasBuiltInSub:       true,
	}
}

// Detect returns true when ~/.claude exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.Home, ".claude"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to claude-code.
// Claude Code manages its own subscription and settings.json — harness-sync
// only manages skills/agents symlinks, CLAUDE.md, and MCP servers.
func (a *Adapter) Render(b *canonical.Bundle) (*adapter.FileSet, error) {
	fs := adapter.NewFileSet()
	base := filepath.Join(a.Home, ".claude")

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
	// Claude Code natively auto-loads ~/.claude/rules/*.md (memory feature), so
	// rules are delivered as their own directory symlink — NOT folded into
	// CLAUDE.md — preserving each rule's optional path-scoping frontmatter.
	fs.Add(adapter.File{
		Dest:          filepath.Join(base, "rules"),
		Kind:          adapter.SymlinkDir,
		SymlinkTarget: filepath.Join(b.Root, "rules"),
	})

	fs.Add(adapter.File{
		Dest:    filepath.Join(base, "CLAUDE.md"),
		Kind:    adapter.RenderedFile,
		Content: []byte(b.InstructionText(name)),
	})

	// Claude Code reads MCP servers from two files on disk:
	//   * ~/.claude.json (live, written by `claude mcp add`)
	//   * ~/.claude/mcp_servers.json (older / fallback location)
	// Write the dedicated file as the canonical destination, AND merge the
	// same map into ~/.claude.json (preserving every other key in that
	// large state file) so both stay in sync.
	mcpMap := common.BuildMCPMapStyled(&b.MCP, common.MCPClaudeStyle)
	overlay := map[string]any{"mcpServers": mcpMap}

	dedicatedPath := filepath.Join(base, "mcp_servers.json")
	dedicatedExisting, err := common.ReadExistingFile(dedicatedPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dedicatedPath, err)
	}
	dedicatedMerged, err := common.MergeJSONKeys(dedicatedExisting, overlay)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    dedicatedPath,
		Kind:    adapter.RenderedFile,
		Content: dedicatedMerged,
	})

	livePath := filepath.Join(a.Home, ".claude.json")
	if _, err := os.Stat(livePath); err == nil {
		liveExisting, err := common.ReadExistingFile(livePath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", livePath, err)
		}
		liveMerged, err := common.MergeJSONKeys(liveExisting, overlay)
		if err != nil {
			return nil, err
		}
		fs.Add(adapter.File{
			Dest:    livePath,
			Kind:    adapter.RenderedFile,
			Content: liveMerged,
			// NoMerge: .claude.json is a live state file (Claude Code rewrites it
			// constantly). The JSON merge above already reconciles at the key level;
			// a git 3-way merge on top causes perpetual conflicts.
			NoMerge: true,
		})
	}

	return fs, nil
}

// Import reads claude-code config from home and returns a canonical ImportResult.
func (a *Adapter) Import(home string) (*adapter.ImportResult, error) {
	return importFrom(home)
}
