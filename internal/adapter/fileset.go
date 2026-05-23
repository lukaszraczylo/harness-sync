package adapter

import "os"

type File struct {
	Dest          string
	SymlinkTarget string
	Content       []byte
	Mode          os.FileMode
	Kind          Kind
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
