package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

func TestMergeManagedMarkdown(t *testing.T) {
	t.Run("empty existing returns just the block", func(t *testing.T) {
		out := MergeManagedMarkdown("", "hello")
		assert.Contains(t, out, ManagedBlockBegin)
		assert.Contains(t, out, "hello")
		assert.Contains(t, out, ManagedBlockEnd)
	})

	t.Run("no markers appends, preserving user content", func(t *testing.T) {
		out := MergeManagedMarkdown("# My notes\nkeep me\n", "managed body")
		assert.Contains(t, out, "# My notes")
		assert.Contains(t, out, "keep me")
		assert.Contains(t, out, "managed body")
	})

	t.Run("existing markers replace only the block, idempotent", func(t *testing.T) {
		first := MergeManagedMarkdown("user above\n", "v1")
		second := MergeManagedMarkdown(first, "v2")
		assert.Contains(t, second, "user above")
		assert.Contains(t, second, "v2")
		assert.NotContains(t, second, "v1")
		// Only one managed block after a re-apply.
		assert.Equal(t, 1, countOccurrences(second, ManagedBlockBegin))

		// Same input → same output (idempotent).
		third := MergeManagedMarkdown(second, "v2")
		assert.Equal(t, second, third)
	})
}

func countOccurrences(s, sub string) int {
	n := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			n++
		}
	}
	return n
}

func TestUnionNestedMap(t *testing.T) {
	existing := []byte(`{"mcpServers":{"user":{"command":"/bin/user"}},"other":1}`)
	ours := map[string]any{"ours": map[string]any{"command": "/bin/ours"}}
	merged := UnionNestedMap(existing, "mcpServers", ours)
	assert.Contains(t, merged, "user", "user-added entry preserved")
	assert.Contains(t, merged, "ours", "our entry added")

	t.Run("ours wins on key collision", func(t *testing.T) {
		ex := []byte(`{"mcp":{"x":{"v":"old"}}}`)
		m := UnionNestedMap(ex, "mcp", map[string]any{"x": map[string]any{"v": "new"}})
		assert.Equal(t, "new", m["x"].(map[string]any)["v"])
	})

	t.Run("empty existing returns ours", func(t *testing.T) {
		m := UnionNestedMap(nil, "mcp", map[string]any{"a": 1})
		assert.Equal(t, map[string]any{"a": 1}, m)
	})
}

func TestMergeProviderMap(t *testing.T) {
	// Existing has a user provider under a different key plus a duplicate of our
	// gateway URL under a stray key.
	existing := []byte(`{"provider":{
		"userprov":{"npm":"x","options":{"baseURL":"https://other"}},
		"stray":{"npm":"y","options":{"baseURL":"https://gw"},"models":{"m1":{"name":"m1"}}}
	}}`)
	ours := map[string]any{
		"hs-gw": map[string]any{"npm": "@ai-sdk/openai-compatible", "options": map[string]any{"baseURL": "https://gw"}},
	}
	merged := MergeProviderMap(existing, ours, "https://gw")
	assert.Contains(t, merged, "userprov", "unrelated user provider preserved")
	assert.Contains(t, merged, "hs-gw", "our provider present")
	assert.NotContains(t, merged, "stray", "duplicate same-URL provider absorbed")
	// The stray's models were absorbed into ours.
	hsModels := merged["hs-gw"].(map[string]any)["models"].(map[string]any)
	assert.Contains(t, hsModels, "m1")
}

func TestGooseAPIKeyEnv(t *testing.T) {
	assert.Equal(t, "CUSTOM_HS_GW_API_KEY", GooseAPIKeyEnv("custom_hs-gw"))
	assert.Equal(t, "CUSTOM_HS_LLMGW_H_RACZYLO_COM_API_KEY", GooseAPIKeyEnv("custom_hs-llmgw-h-raczylo-com"))
}

func TestStripProviderPrefix(t *testing.T) {
	assert.Equal(t, "claude-x", StripProviderPrefix("anthropic/claude-x"))
	assert.Equal(t, "claude-x", StripProviderPrefix("claude-x"))
	assert.Equal(t, "b/c", StripProviderPrefix("a/b/c"))
}

func TestAbsorbDuplicateProvidersPreservesArrayModels(t *testing.T) {
	// A duplicate at the same URL whose models is array-shaped must NOT be
	// dropped (we can't merge an array into our map; keep the duplicate).
	provMap := map[string]any{
		"hs-gw": map[string]any{"options": map[string]any{"baseURL": "https://gw"}},
		"dup":   map[string]any{"options": map[string]any{"baseURL": "https://gw"}, "models": []any{"m1"}},
	}
	out := AbsorbDuplicateProviders(provMap, "hs-gw", "https://gw")
	assert.Contains(t, out, "dup", "array-shaped-models duplicate preserved, not dropped")
}

func TestProvidersAsMapTranslatesEnvPlaceholder(t *testing.T) {
	p := &canonical.Profile{
		Gateway: canonical.Gateway{URL: "https://gw", Token: "${LLM_TOKEN}"},
	}
	m := ProvidersAsMap(p)
	opts := m["hs-gw"].(map[string]any)["options"].(map[string]any)
	assert.Equal(t, "{env:LLM_TOKEN}", opts["apiKey"], "opencode dialect is {env:VAR}, not ${VAR}")

	t.Run("literal token left untouched", func(t *testing.T) {
		p := &canonical.Profile{Gateway: canonical.Gateway{URL: "https://gw", Token: "dummy"}}
		m := ProvidersAsMap(p)
		opts := m["hs-gw"].(map[string]any)["options"].(map[string]any)
		assert.Equal(t, "dummy", opts["apiKey"])
	})
}

func TestProvidersAsCrushMapHonoursAlias(t *testing.T) {
	p := &canonical.Profile{
		Gateway: canonical.Gateway{URL: "https://gw", Token: "t"},
		Models:  []canonical.Model{{ID: "claude-sonnet-4-6", Alias: "sonnet"}},
	}
	m := ProvidersAsCrushMap(p)
	models := m["hs-gw"].(map[string]any)["models"].([]map[string]any)
	require.Len(t, models, 1)
	assert.Equal(t, "claude-sonnet-4-6", models[0]["id"], "wire id stays the model ID")
	assert.Equal(t, "sonnet", models[0]["name"], "label uses the alias")
}

func TestGooseCustomProviderFileSchema(t *testing.T) {
	p := &canonical.Profile{
		Gateway: canonical.Gateway{URL: "https://gw/v1", Token: "tok", DefaultModel: "anthropic/sonnet"},
	}
	body, providerName := GooseCustomProviderFile(p)
	require.NotNil(t, body)
	assert.Equal(t, "custom_hs-gw", providerName)
	var cp map[string]any
	require.NoError(t, json.Unmarshal(body, &cp))
	assert.NotContains(t, cp, "api_key")
	assert.Equal(t, "CUSTOM_HS_GW_API_KEY", cp["api_key_env"])
	assert.Equal(t, true, cp["requires_auth"])
	// /v1 already present → no double chat/completions, single append.
	assert.Equal(t, "https://gw/v1/chat/completions", cp["base_url"])
}
