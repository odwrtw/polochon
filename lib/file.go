package polochon

import (
	"errors"
	"os"
	"path"
	"strings"
)

// File errors
var (
	ErrInvalidGuessType = errors.New("file: invalid guess type")
)

// File handles polochon files
type File struct {
	FileConfig `xml:"-" json:"-"`
	Path       string `xml:"-" json:"-"`
}

// NewFile returs a new file from a path
func NewFile(path string) *File {
	return &File{
		Path: path,
	}
}

// NewFileWithConfig returs a new file from a path
func NewFileWithConfig(path string, conf FileConfig) *File {
	return &File{
		FileConfig: conf,
		Path:       path,
	}
}

// Exists returns true is the file exists
func (f *File) Exists() bool {
	return Exists(f.Path)
}

// IsVideo returns true is the file is considered as a video, using the allowed
// extentions in the configuration
func (f *File) IsVideo() bool {
	// Get the lower case extention
	ext := path.Ext(strings.ToLower(f.Path))

	// Check in the video extentions
	for _, e := range f.VideoExtentions {
		if e == ext {
			return true
		}
	}
	return false
}

// IsIgnored returns true if the file has a ".ignore" file with the same name
func (f *File) IsIgnored() bool {
	if _, err := os.Stat(f.IgnorePath()); err == nil {
		return true
	}
	return false
}

// IsExcluded returns true if the file contains an excluded word
func (f *File) IsExcluded() bool {
	fileName := strings.ToLower(path.Base(f.Path))

	for _, excluded := range f.ExcludeFileContaining {
		if strings.Contains(fileName, excluded) {
			return true
		}
	}
	return false
}

// Ignore create a ".ignore" file next to the file to indicate that it is
// ignored
func (f *File) Ignore() error {
	file, err := os.Create(f.IgnorePath())
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}

// Guess video information from file
func (f *File) Guess(conf VideoConfig) (Video, error) {
	return f.Guesser.Guess(conf, *f)
}

// NfoPath is an helper to get the nfo filename from the video filename
func (f *File) NfoPath() string {
	return f.filePathWithoutExt() + ".nfo"
}

// SubtitlePath is an helper to get the subtitle path from the  filename
func (f *File) SubtitlePath() string {
	return f.filePathWithoutExt() + ".srt"
}

// IgnorePath is an helper to get the ignore file path
func (f *File) IgnorePath() string {
	return f.Path + ".ignore"
}

// MovieFanartPath is an helper to get the movie fanart path from the video path
func (f *File) MovieFanartPath() string {
	return f.filePathWithoutExt() + "-fanart.jpg"
}

// filePathWithoutExt returns the file path without the file extension
func (f *File) filePathWithoutExt() string {
	return RemoveExt(f.Path)
}

// RemoveExt returns file path without the extention
func RemoveExt(filepath string) string {
	// Extention
	ext := path.Ext(filepath)
	// File length without the extension
	l := len(filepath) - len(ext)
	// Rebuild path
	return filepath[:l]
}

// Exists tests if file exists
func Exists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
