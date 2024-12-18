package polochon

import (
	"encoding/binary"
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
	Guessers                  []Guesser

	allowedVideoExt map[string]struct{}
}

// IsVideo returns true if the file is considered a video based on the extension of the file
func (fc *FileConfig) IsVideo(filename string) bool {
	if len(fc.allowedVideoExt) == 0 {
		fc.allowedVideoExt = make(map[string]struct{}, len(fc.VideoExtensions))
		for _, ext := range fc.VideoExtensions {
			fc.allowedVideoExt[ext] = struct{}{}
		}
	}

	// Get the lower case extension
	ext := path.Ext(strings.ToLower(filename))
	_, ok := fc.allowedVideoExt[ext]
	return ok
}

// File handles polochon files
type File struct {
	FileConfig `json:"-"`
	Path       string `json:"-"`
	Size       int64  `json:"size"`
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

// IsVideo returns true is the file is considered to be a video
func (f *File) IsVideo() bool {
	return f.FileConfig.IsVideo(f.Path)
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
	for _, guesser := range f.Guessers {
		v, err := guesser.Guess(*f, movieConf, showConf, log)
		if err == nil {
			return v, err
		}

		if err != ErrNotAvailable {
			log.WithField("guesser", guesser.Name()).Debugf("failed to guess video")
		}
	}
	return nil, ErrGuessingVideo
}

// GuessMetadata guesses the metadata of a file
func (f *File) GuessMetadata(log *logrus.Entry) (*VideoMetadata, error) {
	var updated bool
	m := &VideoMetadata{}
	for _, guesser := range f.Guessers {
		metadata, err := guesser.GuessMetadata(f, log)
		if err == nil {
			updated = true
			m.Update(metadata)
			continue
		}

		if err != ErrNotAvailable {
			log.WithField("guesser", guesser.Name()).Debugf("failed to guess metadata")
		}
	}

	if updated {
		return m, nil
	}

	return nil, ErrGuessingMetadata
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

// Ext returns the file extension
func (f *File) Ext() string {
	return path.Ext(f.Path)
}

// MovieFanartPath returns the movie fanart path
func (f *File) MovieFanartPath() string {
	return f.PathWithoutExt() + "-fanart.jpg"
}

// MovieThumbPath returns the movie thumb path
func (f *File) MovieThumbPath() string {
	return filepath.Join(path.Dir(f.Path), "/poster.jpg")
}

// OpensubHash implements the opensubtitles hash functions:
// https://trac.opensubtitles.org/projects/opensubtitles/wiki/HashSourceCodes
func (f *File) OpensubHash() (uint64, error) {
	const hashChunkSize = 65536 // 64k
	const hashBufSize = 8       // 8 bytes

	if f.Size < hashChunkSize {
		return 0, fmt.Errorf("polochon: file to small to be hashed")
	}

	file, err := os.Open(f.Path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	hash := uint64(f.Size)

	buf := make([]byte, hashBufSize)
	parts := hashChunkSize / 8
	for _, offset := range []int64{0, f.Size - hashChunkSize} {
		_, err := file.Seek(offset, 0)
		if err != nil {
			return 0, err
		}

		for i := 0; i < parts; i++ {
			n, err := file.Read(buf)
			if err != nil {
				return 0, err
			}

			if n != hashBufSize {
				return 0, fmt.Errorf("polochon: failed to read all bytes %d/%d", n, hashBufSize)
			}

			hash += binary.LittleEndian.Uint64(buf)
		}
	}

	return hash, nil
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
