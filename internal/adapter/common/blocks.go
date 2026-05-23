// Package common provides shared helpers for harness adapters.
package common

import "github.com/lukaszraczylo/harness-sync/internal/canonical"

// ProviderEntry is a generic provider record. Adapters may massage further.
type ProviderEntry = map[string]any

// BuildProviders returns the canonical-to-generic provider list: gateway first
// (if configured), then any upstreams. Each entry has id, name, base_url,
// api_key, and (for the gateway) a models list.
func BuildProviders(p *canonical.Profile) []ProviderEntry {
	out := make([]ProviderEntry, 0, len(p.Upstreams)+1)
	if p.Gateway.URL != "" {
		gw := ProviderEntry{
			"id":       "harness-sync-gateway",
			"name":     "harness-sync gateway",
			"base_url": p.Gateway.URL,
			"api_key":  p.Gateway.Token,
		}
		if models := buildModels(p.Models); len(models) > 0 {
			gw["models"] = models
		}
		out = append(out, gw)
	}
	for _, up := range p.Upstreams {
		e := ProviderEntry{"id": up.Name, "name": up.Name}
		if up.BaseURL != "" {
			e["base_url"] = up.BaseURL
		}
		if up.APIKey != "" {
			e["api_key"] = up.APIKey
		}
		out = append(out, e)
	}
	return out
}

func buildModels(models []canonical.Model) []map[string]any {
	out := make([]map[string]any, 0, len(models))
	for _, m := range models {
		e := map[string]any{"id": m.ID}
		if m.Alias != "" {
			e["alias"] = m.Alias
		}
		out = append(out, e)
	}
	return out
}

// BuildMCPMap returns canonical MCP servers as a name -> object map.
// Empty map (not nil) is returned when there are no servers so the caller
// can decide whether to include the key.
func BuildMCPMap(reg *canonical.MCPRegistry) map[string]any {
	out := map[string]any{}
	if reg == nil {
		return out
	}
	for _, s := range reg.Servers {
		e := map[string]any{}
		if s.Command != "" {
			e["command"] = s.Command
		}
		if len(s.Args) > 0 {
			e["args"] = s.Args
		}
		if s.URL != "" {
			e["url"] = s.URL
		}
		if s.Transport != "" {
			e["transport"] = s.Transport
		}
		if len(s.Env) > 0 {
			e["env"] = s.Env
		}
		out[s.Name] = e
	}
	return out
}
