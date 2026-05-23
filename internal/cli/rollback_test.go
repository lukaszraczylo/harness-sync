package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRollbackHelp(t *testing.T) {
	cmd := NewRollback()
	assert.Equal(t, "rollback [n]", cmd.Use)
}
