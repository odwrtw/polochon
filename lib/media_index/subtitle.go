package index

import polochon "github.com/odwrtw/polochon/lib"

// Subtitle represents a subtitle
type Subtitle struct {
	Size int64             `json:"size"`
	Lang polochon.Language `json:"lang"`
}

// NewSubtitle returns a new subtitle from a polochon subtitle
func NewSubtitle(s *polochon.Subtitle) *Subtitle {
	if s == nil {
		return nil
	}

	return &Subtitle{
		Lang: s.Lang,
		Size: s.Size,
	}
}
