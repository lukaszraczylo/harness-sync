package canonical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileActiveModel(t *testing.T) {
	p := &Profile{
		Name: "home",
		Gateway: Gateway{
			URL:          "https://gw",
			Token:        "dummy",
			DefaultModel: "sonnet",
		},
		Models: []Model{
			{ID: "claude-sonnet-4-6", Alias: "sonnet"},
			{ID: "claude-opus-4-7", Alias: "opus"},
		},
	}
	m, ok := p.LookupModel("sonnet")
	assert.True(t, ok)
	assert.Equal(t, "claude-sonnet-4-6", m.ID)

	_, ok = p.LookupModel("missing")
	assert.False(t, ok)
}
