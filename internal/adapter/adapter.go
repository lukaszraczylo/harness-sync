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

type Adapter interface {
	Name() string
	Detect() bool
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
