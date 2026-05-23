package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiSelectNonInteractive(t *testing.T) {
	sel, err := MultiSelect("pick", []string{"a", "b", "c"}, WithNonInteractive([]string{"a", "c"}))
	assert.NoError(t, err)
	assert.Equal(t, []string{"a", "c"}, sel)
}

func TestMultiSelectNonInteractiveInvalidChoice(t *testing.T) {
	_, err := MultiSelect("pick", []string{"a", "b"}, WithNonInteractive([]string{"z"}))
	assert.Error(t, err)
}
