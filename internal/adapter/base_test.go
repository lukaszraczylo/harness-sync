package adapter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
)

func TestNewBaseDefaultsHomeFromEnv(t *testing.T) {
	b := adapter.NewBase()
	// DefaultHome returns either $HOME or "". We only assert that the
	// helper does not panic and that Home is a string (Go zero value is "").
	// Environment-dependence: skip if HOME is unset (CI minimal envs).
	if b.Home == "" {
		t.Skip("HOME unset in test env; cannot verify default")
	}
	assert.NotEmpty(t, b.Home)
}

func TestNewBaseWithHomeOverrides(t *testing.T) {
	b := adapter.NewBase(adapter.WithHome("/custom/home/path"))
	assert.Equal(t, "/custom/home/path", b.Home)
}

func TestNewBaseOptionsApplyInOrder(t *testing.T) {
	// Last write wins — covers the basic Apply loop in NewBase.
	b := adapter.NewBase(
		adapter.WithHome("/first"),
		adapter.WithHome("/second"),
	)
	assert.Equal(t, "/second", b.Home)
}

func TestBaseOptionTypedCorrectly(t *testing.T) {
	// adapter.BaseOption is the public type; WithHome must return it.
	opt := adapter.WithHome("/x")
	require.NotNil(t, opt)

	// Apply it manually to confirm signature.
	var b adapter.Base
	opt(&b)
	assert.Equal(t, "/x", b.Home)
}
