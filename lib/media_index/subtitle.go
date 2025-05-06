package index

import (
	polochon "github.com/odwrtw/polochon/lib"
)

// Subtitle represents a subtitle
type Subtitle struct {
	Embedded bool              `json:"embedded"`
	Size     int64             `json:"size"`
	Lang     polochon.Language `json:"lang"`
}

// NewSubtitle returns a new subtitle from a polochon subtitle
func NewSubtitle(s *polochon.Subtitle) *Subtitle {
	if s == nil {
		return nil
	}

	return &Subtitle{
		Embedded: s.Embedded,
		Lang:     s.Lang,
		Size:     s.Size,
	}
}

func upsertSubtitle(subs []*Subtitle, sub *Subtitle) []*Subtitle {
	if sub == nil {
		return subs
	}

	if subs == nil {
		return []*Subtitle{sub}
	}

	idx := -1
	newSubs := subs
	for i, oldSub := range subs {
		if oldSub.Embedded {
			continue
		}

		if sub.Lang == oldSub.Lang {
			idx = i
			break
		}
	}

	if idx >= 0 {
		newSubs[idx] = sub
	} else {
		newSubs = append(subs, sub)
	}

	return newSubs
}
