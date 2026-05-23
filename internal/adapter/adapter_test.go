package adapter

import (
	"testing"

	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/stretchr/testify/assert"
)

func TestFileSetAddAndIterate(t *testing.T) {
	fs := NewFileSet()
	fs.Add(File{Dest: "/tmp/a", Kind: RenderedFile, Content: []byte("a")})
	fs.Add(File{Dest: "/tmp/b", Kind: SymlinkFile, SymlinkTarget: "/canon/b"})

	seen := map[string]Kind{}
	fs.ForEach(func(f File) {
		seen[f.Dest] = f.Kind
	})
	assert.Equal(t, RenderedFile, seen["/tmp/a"])
	assert.Equal(t, SymlinkFile, seen["/tmp/b"])
}

func TestKindString(t *testing.T) {
	assert.Equal(t, "rendered", RenderedFile.String())
	assert.Equal(t, "symlink-file", SymlinkFile.String())
}

type fakeAdapter struct {
	name   string
	detect bool
}

func (f *fakeAdapter) Name() string                                 { return f.name }
func (f *fakeAdapter) Detect() bool                                 { return f.detect }
func (f *fakeAdapter) Render(_ *canonical.Bundle) (*FileSet, error) { return NewFileSet(), nil }
func (f *fakeAdapter) Import(_ string) (*ImportResult, error)       { return &ImportResult{}, nil }

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	a := &fakeAdapter{name: "a", detect: true}
	b := &fakeAdapter{name: "b", detect: false}
	r.Register(a)
	r.Register(b)

	assert.Equal(t, []string{"a", "b"}, r.Names())
	assert.Equal(t, []string{"a"}, r.DetectedNames())

	got, ok := r.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "a", got.Name())

	_, ok = r.Get("missing")
	assert.False(t, ok)
}

func TestRegistryDuplicatePanics(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeAdapter{name: "x"})
	assert.Panics(t, func() { r.Register(&fakeAdapter{name: "x"}) })
}
