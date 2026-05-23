package merge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanFastForward(t *testing.T) {
	res, err := ThreeWay(Inputs{
		Base:   []byte("a\nb\nc\n"),
		Ours:   []byte("a\nb\nc\nd\n"),
		Theirs: []byte("a\nb\nc\n"),
	})
	require.NoError(t, err)
	assert.False(t, res.Conflict)
	assert.Equal(t, "a\nb\nc\nd\n", string(res.Body))
}

func TestConflict(t *testing.T) {
	res, err := ThreeWay(Inputs{
		Base:   []byte("a\n"),
		Ours:   []byte("a\nours\n"),
		Theirs: []byte("a\ntheirs\n"),
	})
	require.NoError(t, err)
	assert.True(t, res.Conflict)
	assert.Contains(t, string(res.Body), "<<<<<<<")
	assert.Contains(t, string(res.Body), ">>>>>>>")
}

func TestNoOpWhenAllEqual(t *testing.T) {
	res, err := ThreeWay(Inputs{
		Base:   []byte("a\n"),
		Ours:   []byte("a\n"),
		Theirs: []byte("a\n"),
	})
	require.NoError(t, err)
	assert.False(t, res.Conflict)
	assert.Equal(t, "a\n", string(res.Body))
}
