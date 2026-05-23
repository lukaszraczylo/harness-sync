// Package common provides shared helpers for harness adapters.
package common

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

const fallbackGatewayProviderID = "harness-sync-gateway"

// GatewayProviderID is the canonical provider key used across all adapters.
// Deprecated: prefer GatewayProviderKey(gatewayURL) for a stable URL-derived key.
const GatewayProviderID = fallbackGatewayProviderID

// GatewayProviderKey returns a stable provider key derived from the gateway URL.
// Format: "hs-<hostname>" with dots and special chars normalised to dashes.
// Falls back to "harness-sync-gateway" when the URL is empty or unparseable.
//
//	https://llmgw.h.raczylo.com/v1  →  hs-llmgw-h-raczylo-com
func GatewayProviderKey(gatewayURL string) string {
	if gatewayURL == "" {
		return fallbackGatewayProviderID
	}
	u, err := url.Parse(gatewayURL)
	if err != nil || u.Hostname() == "" {
		return fallbackGatewayProviderID
	}
	host := u.Hostname()
	// Replace dots, underscores, and colons (port separators) with dashes.
	r := strings.NewReplacer(".", "-", "_", "-", ":", "-")
	return "hs-" + r.Replace(host)
}

// ProviderEntry is a generic provider record. Adapters may massage further.
type ProviderEntry = map[string]any

// BuildProviders returns the canonical-to-generic provider list: gateway first
// (if configured), then any upstreams. Each entry has id, name, base_url,
// api_key, and (for the gateway) a models list.
func BuildProviders(p *canonical.Profile) []ProviderEntry {
	out := make([]ProviderEntry, 0, len(p.Upstreams)+1)
	if p.Gateway.URL != "" {
		key := GatewayProviderKey(p.Gateway.URL)
		gw := ProviderEntry{
			"id":       key,
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

// crushModel builds a 10-field model entry required by the crush schema.
func crushModel(id string) map[string]any {
	return map[string]any{
		"id":                     id,
		"name":                   id,
		"cost_per_1m_in":         0,
		"cost_per_1m_out":        0,
		"cost_per_1m_in_cached":  0,
		"cost_per_1m_out_cached": 0,
		"context_window":         200000,
		"default_max_tokens":     8192,
		"can_reason":             false,
		"supports_attachments":   true,
	}
}

// ProvidersAsCrushMap returns a map[providerID]providerObject for the crush
// harness. Each provider has type, base_url, api_key, and models[].
// models[] entries carry all 10 required crush schema fields.
func ProvidersAsCrushMap(p *canonical.Profile) map[string]any {
	out := map[string]any{}
	if p.Gateway.URL == "" {
		return out
	}
	models := make([]map[string]any, 0)
	for _, m := range p.Models {
		models = append(models, crushModel(m.ID))
	}
	if len(models) == 0 && p.Gateway.DefaultModel != "" {
		models = append(models, crushModel(p.Gateway.DefaultModel))
	}
	out[GatewayProviderKey(p.Gateway.URL)] = map[string]any{
		"type":     "openai-compat",
		"base_url": p.Gateway.URL,
		"api_key":  p.Gateway.Token,
		"models":   models,
	}
	return out
}

// CrushRoleModels returns the top-level models map for crush with large, small,
// and title roles all pointing at the gateway provider + default model.
func CrushRoleModels(p *canonical.Profile) map[string]any {
	if p.Gateway.DefaultModel == "" {
		return map[string]any{}
	}
	sel := map[string]any{
		"model":    p.Gateway.DefaultModel,
		"provider": GatewayProviderKey(p.Gateway.URL),
	}
	return map[string]any{
		"large": sel,
		"small": sel,
		"title": sel,
	}
}

// ProvidersAsMap returns a name-keyed map of provider objects for harnesses
// like opencode/kilo that use `provider` (singular, map-shaped). Includes
// top-level "name" and "models" Record so the harness UI can enumerate models.
func ProvidersAsMap(p *canonical.Profile) map[string]any {
	out := map[string]any{}
	if p.Gateway.URL != "" {
		models := map[string]any{}
		for _, m := range p.Models {
			displayName := m.ID
			if m.Alias != "" {
				displayName = m.Alias
			}
			models[m.ID] = map[string]any{
				"name":  displayName,
				"limit": map[string]any{"context": 200000, "output": 8192},
			}
		}
		entry := map[string]any{
			"name": "harness-sync gateway",
			"npm":  "@ai-sdk/openai-compatible",
			"options": map[string]any{
				"baseURL": p.Gateway.URL,
				"apiKey":  p.Gateway.Token,
			},
		}
		if len(models) > 0 {
			entry["models"] = models
		}
		out[GatewayProviderKey(p.Gateway.URL)] = entry
	}
	for _, up := range p.Upstreams {
		entry := map[string]any{
			"npm": "@ai-sdk/openai-compatible",
		}
		opts := map[string]any{}
		if up.BaseURL != "" {
			opts["baseURL"] = up.BaseURL
		}
		if up.APIKey != "" {
			opts["apiKey"] = up.APIKey
		}
		if len(opts) > 0 {
			entry["options"] = opts
		}
		out[up.Name] = entry
	}
	return out
}

// KiloModelString returns the "providerID/modelID" string used by kilo/opencode.
func KiloModelString(p *canonical.Profile) string {
	if p.Gateway.DefaultModel == "" {
		return ""
	}
	return GatewayProviderKey(p.Gateway.URL) + "/" + p.Gateway.DefaultModel
}

// ProvidersAsCagentMap returns the providers block for cagent:
// map[providerID]{base_url, token_key, provider}.
func ProvidersAsCagentMap(p *canonical.Profile) map[string]any {
	out := map[string]any{}
	if p.Gateway.URL == "" {
		return out
	}
	out[GatewayProviderKey(p.Gateway.URL)] = map[string]any{
		"base_url": p.Gateway.URL,
		"api_key":  p.Gateway.Token,
		"provider": "openai",
	}
	return out
}

// ZedLanguageModels returns the language_models.openai block for Zed when a
// gateway URL is configured.
func ZedLanguageModels(p *canonical.Profile) map[string]any {
	if p.Gateway.URL == "" {
		return map[string]any{}
	}
	models := make([]map[string]any, 0)
	for _, m := range p.Models {
		models = append(models, map[string]any{
			"name": m.ID,
			"display_name": func() string {
				if m.Alias != "" {
					return m.Alias
				}
				return m.ID
			}(),
			"max_tokens": 8192,
		})
	}
	if len(models) == 0 && p.Gateway.DefaultModel != "" {
		models = append(models, map[string]any{
			"name":         p.Gateway.DefaultModel,
			"display_name": p.Gateway.DefaultModel,
			"max_tokens":   8192,
		})
	}
	return map[string]any{
		"openai": map[string]any{
			"api_url":          p.Gateway.URL,
			"available_models": models,
		},
	}
}

// GooseCustomProviderFile returns the JSON body and provider ID for the goose
// custom_providers/<id>.json file. Returns nil body when Gateway.URL is empty.
func GooseCustomProviderFile(p *canonical.Profile) ([]byte, string) {
	if p.Gateway.URL == "" {
		return nil, ""
	}
	providerID := GatewayProviderKey(p.Gateway.URL)
	providerName := "custom_" + providerID

	models := make([]map[string]any, 0)
	for _, m := range p.Models {
		models = append(models, map[string]any{
			"name":          m.ID,
			"context_limit": 200000,
		})
	}
	if len(models) == 0 && p.Gateway.DefaultModel != "" {
		// strip optional "provider/" prefix from DefaultModel
		modelID := p.Gateway.DefaultModel
		if idx := strings.IndexByte(modelID, '/'); idx >= 0 {
			modelID = modelID[idx+1:]
		}
		models = append(models, map[string]any{
			"name":          modelID,
			"context_limit": 200000,
		})
	}

	// display_name: hostname of the gateway URL, or just providerID.
	displayName := providerID
	if idx := strings.Index(p.Gateway.URL, "://"); idx >= 0 {
		rest := p.Gateway.URL[idx+3:]
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			rest = rest[:slash]
		}
		if rest != "" {
			displayName = rest
		}
	}

	// Append only /chat/completions — the profile URL already contains /v1.
	chatURL := strings.TrimRight(p.Gateway.URL, "/") + "/chat/completions"

	entry := map[string]any{
		"name":         providerName,
		"engine":       "openai",
		"display_name": displayName,
		"base_url":     chatURL,
		"models":       models,
		"api_key":      p.Gateway.Token,
	}
	body, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return nil, ""
	}
	return body, providerName
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
