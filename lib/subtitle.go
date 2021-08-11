package polochon

import (
	"bytes"
	"errors"
	"io"
	"os"
)

// Subtitle errors
var (
	ErrMissingSubtitleLang = errors.New("polochon: no subtitle lang")
	ErrMissingSubtitlePath = errors.New("polochon: no subtitle path")
)

// Subtitle represents a subtitle
type Subtitle struct {
	File

	Data []byte

	Lang  Language `json:"lang"`
	Video Video    `json:"-"`
}

// NewSubtitleFromVideo returns a subtitle from a video
func NewSubtitleFromVideo(v Video, l Language) *Subtitle {
	file := NewFile(v.GetFile().SubtitlePath(l))
	return &Subtitle{
		File:  *file,
		Lang:  l,
		Video: v,
	}
}

// Save saves the subtitle to its path
func (s *Subtitle) Save() error {
	if s.Lang == "" {
		return ErrMissingSubtitleLang
	}

	if s.Video != nil && s.Video.GetFile() != nil {
		// Update the path of the subtitle according to the video path, this is
		// usefull if the video as been moved
		s.Path = s.Video.GetFile().SubtitlePath(s.Lang)
	}

	if s.Path == "" {
		return ErrMissingSubtitlePath
	}

	file, err := os.Create(s.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	i, err := io.Copy(file, bytes.NewReader(s.Data))
	s.Size = i

	return err
}
