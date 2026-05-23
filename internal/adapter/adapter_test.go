package adapter

import (
	"testing"

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
