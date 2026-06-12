package common

import "github.com/lukaszraczylo/harness-sync/internal/canonical"

// MCPStyle selects the dialect for MCP server entries.
type MCPStyle int

const (
	// MCPClaudeStyle: claude-code shape — {command, args, env, url, transport}.
	MCPClaudeStyle MCPStyle = iota
	// MCPCrushStyle: crush shape — adds type: stdio|http|sse.
	MCPCrushStyle
	// MCPOpencodeStyle: kilo + opencode shape — type: local|remote, command as []string.
	MCPOpencodeStyle
	// MCPZedStyle: zed context_servers — {enabled, command, args} or {enabled, url}.
	MCPZedStyle
)

// BuildMCPMapStyled returns the MCP map for the given harness dialect.
func BuildMCPMapStyled(reg *canonical.MCPRegistry, style MCPStyle) map[string]any {
	out := map[string]any{}
	if reg == nil {
		return out
	}
	for _, s := range reg.Servers {
		out[s.Name] = mcpEntry(s, style)
	}
	return out
}

func mcpEntry(s canonical.MCPServer, style MCPStyle) map[string]any {
	e := map[string]any{}
	switch style {
	case MCPClaudeStyle, MCPCrushStyle:
		if s.URL != "" {
			t := s.Transport
			if t == "" {
				t = "http"
			}
			e["type"] = t
			e["url"] = s.URL
			if len(s.Headers) > 0 {
				e["headers"] = s.Headers
			}
		} else {
			// Claude honours Transport field; Crush always uses "stdio".
			t := "stdio"
			if style == MCPClaudeStyle && s.Transport != "" {
				t = s.Transport
			}
			e["type"] = t
			if s.Command != "" {
				e["command"] = s.Command
			}
			if len(s.Args) > 0 {
				e["args"] = s.Args
			}
			if len(s.Env) > 0 {
				e["env"] = s.Env
			}
		}
	case MCPOpencodeStyle:
		if s.URL != "" {
			e["type"] = "remote"
			e["url"] = s.URL
			if len(s.Headers) > 0 {
				e["headers"] = s.Headers
			}
		} else {
			e["type"] = "local"
			cmd := []string{}
			if s.Command != "" {
				cmd = append(cmd, s.Command)
			}
			cmd = append(cmd, s.Args...)
			e["command"] = cmd
			if len(s.Env) > 0 {
				e["environment"] = s.Env
			}
		}
		e["enabled"] = true
	case MCPZedStyle:
		// Do NOT emit "enabled" — Zed 1.3.x has a serde flatten+deny_unknown_fields
		// bug where outer-struct fields bleed into the untagged enum variants and
		// fail all of them. Omitting "enabled" defaults to true.
		if s.URL != "" {
			e["url"] = s.URL
			if len(s.Headers) > 0 {
				e["headers"] = s.Headers
			}
		} else {
			if s.Command != "" {
				e["command"] = s.Command
			}
			if len(s.Args) > 0 {
				e["args"] = s.Args
			}
		}
	}
	return e
}
