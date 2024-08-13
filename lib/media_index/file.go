package index

import (
	"os"
	"path/filepath"

	polochon "github.com/odwrtw/polochon/lib"
)

// File represents a metadata file
type File struct {
	Name  string         `json:"name"`
	Size  int64          `json:"size"`
	Video polochon.Video `json:"-"`
}

func newFile(path string) *File {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	return &File{
		Name: filepath.Base(path),
		Size: info.Size(),
	}
}
