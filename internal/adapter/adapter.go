// Package adapter defines the harness adapter interface and FileSet.
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

// HarnessCapabilities declares what an adapter manages.
// HasBuiltInSub means the harness has its own AI subscription (e.g. Claude Max)
// and harness-sync should not write provider/model/endpoint config for it.
type HarnessCapabilities struct {
	ManagesProviders    bool // writes provider/gateway endpoint config
	ManagesModels       bool // writes model selection config
	ManagesMCP          bool // writes MCP server config
	ManagesSkills       bool // writes skills symlink or equivalent
	ManagesInstructions bool // writes instructions file (CLAUDE.md, AGENTS.md, etc.)
	HasBuiltInSub       bool // has its own subscription; provider/model config is skipped
}

type Adapter interface {
	Name() string
	Detect() bool
	Capabilities() HarnessCapabilities
	Render(b *canonical.Bundle) (*FileSet, error)
	Import(home string) (*ImportResult, error)
}

type ImportResult struct {
	Instructions string
	Skills       []canonical.Skill
	Agents       []canonical.Agent
	MCP          []canonical.MCPServer
	Profiles     []canonical.Profile
}
