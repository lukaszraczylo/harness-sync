package canonical

import (
	"fmt"

	"github.com/lukaszraczylo/harness-sync/internal/secrets"
)

// SubstituteSecrets walks the bundle and replaces ${VAR} placeholders in all
// config fields using lookup. Skill/agent/instruction bodies are intentionally
// NOT substituted — they are content, not config.
//
// Returns an error (wrapping the offending field path) if any referenced
// variable is missing from lookup.
func SubstituteSecrets(b *Bundle, lookup secrets.Lookup) error {
	var err error

	// Gateway
	if b.Profile.Gateway.Token, err = sub(lookup, "profile.gateway.token", b.Profile.Gateway.Token); err != nil {
		return err
	}
	if b.Profile.Gateway.URL, err = sub(lookup, "profile.gateway.url", b.Profile.Gateway.URL); err != nil {
		return err
	}
	if b.Profile.Gateway.DefaultModel, err = sub(lookup, "profile.gateway.default_model", b.Profile.Gateway.DefaultModel); err != nil {
		return err
	}

	// Upstreams
	for i := range b.Profile.Upstreams {
		u := &b.Profile.Upstreams[i]
		field := fmt.Sprintf("profile.upstreams[%d](%s)", i, u.Name)
		if u.APIKey, err = sub(lookup, field+".api_key", u.APIKey); err != nil {
			return err
		}
		if u.BaseURL, err = sub(lookup, field+".base_url", u.BaseURL); err != nil {
			return err
		}
	}

	// MCP servers
	for i := range b.MCP.Servers {
		sv := &b.MCP.Servers[i]
		field := fmt.Sprintf("mcp.servers[%d](%s)", i, sv.Name)

		if sv.Command, err = sub(lookup, field+".command", sv.Command); err != nil {
			return err
		}
		if sv.URL, err = sub(lookup, field+".url", sv.URL); err != nil {
			return err
		}
		for j, arg := range sv.Args {
			if sv.Args[j], err = sub(lookup, fmt.Sprintf("%s.args[%d]", field, j), arg); err != nil {
				return err
			}
		}
		for k, v := range sv.Env {
			if sv.Env[k], err = sub(lookup, fmt.Sprintf("%s.env[%s]", field, k), v); err != nil {
				return err
			}
		}
	}

	return nil
}

// sub applies secrets.Substitute and wraps errors with field context.
func sub(lookup secrets.Lookup, field, value string) (string, error) {
	result, err := secrets.Substitute(value, lookup)
	if err != nil {
		return "", fmt.Errorf("%s: %w", field, err)
	}
	return result, nil
}
