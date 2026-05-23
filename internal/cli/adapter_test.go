package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func TestAdapterList(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.Register(&detectableAdapter{name: "alpha", detect: true})
	reg.Register(&detectableAdapter{name: "beta", detect: false})

	var buf bytes.Buffer
	cmd := NewAdapter(reg)
	cmd.SetArgs([]string{"list"})
	cmd.SetOut(&buf)
	require.NoError(t, cmd.Execute())

	out := buf.String()
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "beta")
}
