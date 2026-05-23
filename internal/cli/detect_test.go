package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
)

type detectableAdapter struct {
	name   string
	detect bool
}

func (d *detectableAdapter) Name() string { return d.name }
func (d *detectableAdapter) Detect() bool { return d.detect }
func (d *detectableAdapter) Render(_ *canonical.Bundle) (*adapter.FileSet, error) {
	return adapter.NewFileSet(), nil
}
func (d *detectableAdapter) Import(_ string) (*adapter.ImportResult, error) {
	return &adapter.ImportResult{}, nil
}
func (d *detectableAdapter) Capabilities() adapter.HarnessCapabilities {
	return adapter.HarnessCapabilities{ManagesMCP: true}
}

func TestDetectCommandPrintsList(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.Register(&detectableAdapter{name: "yes", detect: true})
	reg.Register(&detectableAdapter{name: "no", detect: false})

	var buf bytes.Buffer
	cmd := NewDetect(reg)
	cmd.SetOut(&buf)
	require.NoError(t, cmd.Execute())

	out := buf.String()
	assert.Contains(t, out, "yes")
	assert.Contains(t, out, "no")
	assert.Contains(t, out, "detected")
	assert.Contains(t, out, "not detected")
}
