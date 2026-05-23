package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONIndent(t *testing.T) {
	b, err := JSON(map[string]any{"a": 1, "b": "x"})
	require.NoError(t, err)
	s := string(b)
	assert.Contains(t, s, "\"a\": 1")
	assert.True(t, s[len(s)-1] == '\n', "trailing newline required")
}

func TestYAML(t *testing.T) {
	b, err := YAML(map[string]any{"a": 1})
	require.NoError(t, err)
	assert.Contains(t, string(b), "a: 1")
}

func TestTOML(t *testing.T) {
	b, err := TOML(map[string]any{"a": 1})
	require.NoError(t, err)
	assert.Contains(t, string(b), "a = 1")
}
