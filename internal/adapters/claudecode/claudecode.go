// Package claudecode is the claude-code harness adapter.
package claudecode

import (
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapter/common"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const name = "claude-code"

// Adapter implements adapter.Adapter for the claude-code harness.
type Adapter struct {
	home string
}

// Option configures an Adapter.
type Option func(*Adapter)

// WithHome overrides the home directory (defaults to os.UserHomeDir).
func WithHome(h string) Option { return func(a *Adapter) { a.home = h } }

// New returns a new Adapter with the given options applied.
func New(opts ...Option) *Adapter {
	a := &Adapter{home: common.DefaultHome()}
	for _, o := range opts {
		o(a)
	}
	return a
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
		ManagesInstructions: true,
		HasBuiltInSub:       true,
	}
}

// Detect returns true when ~/.claude exists.
func (a *Adapter) Detect() bool {
	_, err := os.Stat(filepath.Join(a.home, ".claude"))
	return err == nil
}

// Render produces the FileSet that applies the canonical bundle to claude-code.
// Claude Code manages its own subscription and settings.json — harness-sync
// only manages skills/agents symlinks, CLAUDE.md, and MCP servers.
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

	// Claude Code reads MCP servers from two files on disk:
	//   * ~/.claude.json (live, written by `claude mcp add`)
	//   * ~/.claude/mcp_servers.json (older / fallback location)
	// Write the dedicated file as the canonical destination, AND merge the
	// same map into ~/.claude.json (preserving every other key in that
	// large state file) so both stay in sync.
	mcpMap := common.BuildMCPMapStyled(&b.MCP, common.MCPClaudeStyle)
	overlay := map[string]any{"mcpServers": mcpMap}

	dedicatedPath := filepath.Join(base, "mcp_servers.json")
	dedicatedExisting, _ := os.ReadFile(dedicatedPath)
	dedicatedMerged, err := common.MergeJSONKeys(dedicatedExisting, overlay)
	if err != nil {
		return nil, err
	}
	fs.Add(adapter.File{
		Dest:    dedicatedPath,
		Kind:    adapter.RenderedFile,
		Content: dedicatedMerged,
	})

	livePath := filepath.Join(a.home, ".claude.json")
	if _, err := os.Stat(livePath); err == nil {
		liveExisting, _ := os.ReadFile(livePath)
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
