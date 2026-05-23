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

func TestOSEnvPassThroughOnMissing(t *testing.T) {
	t.Setenv("HARNESS_SYNC_TEST_SET", "value-set")
	v, ok := OSEnv("HARNESS_SYNC_TEST_SET")
	assert.True(t, ok)
	assert.Equal(t, "value-set", v)

	// Unset variable returns the literal placeholder so downstream configs
	// preserve runtime-resolved references like ${CLAUDE_PROJECT}.
	v, ok = OSEnv("HARNESS_SYNC_TEST_NEVER_SET_XYZ")
	assert.True(t, ok)
	assert.Equal(t, "${HARNESS_SYNC_TEST_NEVER_SET_XYZ}", v)
}

func TestStrictOSEnvErrorsOnMissing(t *testing.T) {
	t.Setenv("HARNESS_SYNC_TEST_SET2", "x")
	v, ok := StrictOSEnv("HARNESS_SYNC_TEST_SET2")
	assert.True(t, ok)
	assert.Equal(t, "x", v)

	_, ok = StrictOSEnv("HARNESS_SYNC_TEST_NEVER_SET_XYZ2")
	assert.False(t, ok)
}

func TestSubstituteWithPassThroughOSEnv(t *testing.T) {
	out, err := Substitute("token=${HARNESS_SYNC_NEVER_SET}", OSEnv)
	assert.NoError(t, err)
	assert.Equal(t, "token=${HARNESS_SYNC_NEVER_SET}", out)
}
