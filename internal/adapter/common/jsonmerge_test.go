package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeJSONKeysPreservesOtherKeys(t *testing.T) {
	existing := []byte(`{"a":1,"b":2}`)
	overlay := map[string]any{"b": 3, "c": 4}
	out, err := MergeJSONKeys(existing, overlay)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	assert.Equal(t, float64(1), m["a"])
	assert.Equal(t, float64(3), m["b"])
	assert.Equal(t, float64(4), m["c"])
}

func TestMergeJSONKeysEmptyExisting(t *testing.T) {
	overlay := map[string]any{"x": "y"}
	out, err := MergeJSONKeys(nil, overlay)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	assert.Equal(t, "y", m["x"])
	assert.Len(t, m, 1)
}

func TestMergeJSONKeysInvalidExisting(t *testing.T) {
	existing := []byte(`not json at all`)
	overlay := map[string]any{"k": "v"}
	out, err := MergeJSONKeys(existing, overlay)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	assert.Equal(t, "v", m["k"])
	assert.Len(t, m, 1)
}

func TestMergeJSONKeysNilOverlayValueDeletesKey(t *testing.T) {
	existing := []byte(`{"a":1,"b":2}`)
	overlay := map[string]any{"b": nil}
	out, err := MergeJSONKeys(existing, overlay)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	assert.Equal(t, float64(1), m["a"])
	assert.NotContains(t, m, "b")
}

func TestMergeJSONKeysJSONC(t *testing.T) {
	existing := []byte(`// comment
{
  "a": 1, // trailing
  /* block */ "b": 2
}`)
	overlay := map[string]any{"c": 3}
	out, err := MergeJSONKeys(existing, overlay)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))
	assert.Equal(t, float64(1), m["a"])
	assert.Equal(t, float64(2), m["b"])
	assert.Equal(t, float64(3), m["c"])
}

func TestMergeJSONKeysTrailingNewline(t *testing.T) {
	out, err := MergeJSONKeys(nil, map[string]any{"k": "v"})
	require.NoError(t, err)
	assert.Equal(t, '\n', rune(out[len(out)-1]))
}
