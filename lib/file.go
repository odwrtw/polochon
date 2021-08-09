package polochon

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// FileConfig represents the configuration for a file
type FileConfig struct {
	ExcludeFileContaining     []string
	VideoExtensions           []string
	AllowedExtensionsToDelete []string
	Guesser                   Guesser
}

// File handles polochon files
type File struct {
	FileConfig
	Path string
	Size int64
}

// NewFile returns a new file from a path
func NewFile(path string) *File {
	var size int64 = 0

	info, err := os.Stat(path)
	if err == nil {
		size = info.Size()
	}

	return &File{
		Path: path,
		Size: size,
	}
}

// NewFileWithConfig returns a new file from a path
func NewFileWithConfig(path string, conf FileConfig) *File {
	f := NewFile(path)
	f.FileConfig = conf
	return f
}

// Exists returns true is the file exists
func (f *File) Exists() bool {
	return exists(f.Path)
}

// IsVideo returns true is the file is considered as a video, using the
// allowed extensions in the configuration
func (f *File) IsVideo() bool {
	// Get the lower case extension
	ext := path.Ext(strings.ToLower(f.Path))

	// Check in the video extensions
	for _, e := range f.VideoExtensions {
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

// IsSymlink returns true if the file is a symlink
func (f *File) IsSymlink() bool {
	s, err := os.Lstat(f.Path)
	if err != nil {
		return false
	}

	// Check if it's a symlink
	if s.Mode()&os.ModeSymlink != 0 {
		return true
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
func (f *File) Guess(movieConf MovieConfig, showConf ShowConfig, log *logrus.Entry) (Video, error) {
	return f.Guesser.Guess(*f, movieConf, showConf, log)
}

// GuessMetadata guesses the metadata of a file
func (f *File) GuessMetadata() (*VideoMetadata, error) {
	return f.Guesser.GuessMetadata(f)
}

// NfoPath is an helper to get the nfo filename from the video filename
func (f *File) NfoPath() string {
	return f.PathWithoutExt() + ".nfo"
}

// SubtitlePath is an helper to get the subtitle path from the  filename
func (f *File) SubtitlePath(lang Language) string {
	return fmt.Sprintf("%s.%s.srt", f.PathWithoutExt(), lang.ShortForm())
}

// IgnorePath is an helper to get the ignore file path
func (f *File) IgnorePath() string {
	return f.Path + ".ignore"
}

// Filename returns the file name
func (f *File) Filename() string {
	return filepath.Base(f.Path)
}

// PathWithoutExt returns the file path without the file extension
func (f *File) PathWithoutExt() string {
	return removeExt(f.Path)
}

// MovieFanartPath returns the movie fanart path
func (f *File) MovieFanartPath() string {
	return f.PathWithoutExt() + "-fanart.jpg"
}

// MovieThumbPath returns the movie thumb path
func (f *File) MovieThumbPath() string {
	return filepath.Join(path.Dir(f.Path), "/poster.jpg")
}

// removeExt returns file path without the extension
func removeExt(filepath string) string {
	// Extension
	ext := path.Ext(filepath)
	// File length without the extension
	l := len(filepath) - len(ext)
	// Rebuild path
	return filepath[:l]
}

// Exists tests if file exists
func exists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
