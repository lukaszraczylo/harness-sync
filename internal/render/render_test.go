package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAML(t *testing.T) {
	b, err := YAML(map[string]any{"a": 1})
	require.NoError(t, err)
	assert.Contains(t, string(b), "a: 1")
}
