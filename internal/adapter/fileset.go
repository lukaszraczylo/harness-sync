package adapter

import "os"

type File struct {
	Dest          string
	SymlinkTarget string
	Content       []byte
	Kind          Kind
	Mode          os.FileMode
	// NoMerge skips the 3-way git merge and always writes the rendered content.
	// Use for files the adapter partially manages via JSON/YAML merge (e.g. live
	// state files where the adapter already reconciles at the key level).
	NoMerge bool
}

type FileSet struct {
	files []File
}

func NewFileSet() *FileSet { return &FileSet{} }

func (fs *FileSet) Add(f File) {
	if f.Mode == 0 && f.Kind == RenderedFile {
		f.Mode = 0o644
	}
	fs.files = append(fs.files, f)
}

func (fs *FileSet) ForEach(fn func(File)) {
	for _, f := range fs.files {
		fn(f)
	}
}

func (fs *FileSet) Len() int { return len(fs.files) }
