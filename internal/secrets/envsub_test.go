package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubstituteSimple(t *testing.T) {
	lookup := func(k string) (string, bool) {
		if k == "FOO" {
			return "bar", true
		}
		return "", false
	}
	out, err := Substitute("hello ${FOO}", lookup)
	assert.NoError(t, err)
	assert.Equal(t, "hello bar", out)
}

func TestSubstituteMissingStrict(t *testing.T) {
	lookup := func(k string) (string, bool) { return "", false }
	_, err := Substitute("${MISSING}", lookup)
	assert.Error(t, err)
}

func TestSubstituteEscaped(t *testing.T) {
	lookup := func(k string) (string, bool) { return "should-not-see", true }
	out, err := Substitute("$${LITERAL}", lookup)
	assert.NoError(t, err)
	assert.Equal(t, "${LITERAL}", out)
}
