package mkvinfo

import (
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
)

var videoCodecMap = map[string]string{
	"V_MPEGH/ISO/HEVC": "H.265",
	"V_MPEG4/ISO/AVC":  "H.264",
}

var audioCodecMap = map[string]string{
	"A_AAC":  "AAC",
	"A_AC3":  "Dolby Digital",
	"A_EAC3": "Dolby Digital Plus",
}

var subtitleLangMap = map[string]polochon.Language{
	"fre": polochon.FR,
	"eng": polochon.EN,
}

// Sometimes the first tracks don't have a lang but they have a name starting
// with the lang.
// I've never seen this behaviour for french, that's why it's only for the
// english language.
var subtitlePrefixEnglish = []string{
	"English",
	"US",
	"EN",
}

// TrackType represents the type of a track
type TrackType int64

// TrackTypes
const (
	TrackTypeVideo    TrackType = 1
	TrackTypeAudio    TrackType = 2
	TrackTypeSubtitle TrackType = 17
)

// TrackEntry represents a mkv track entry
type TrackEntry struct {
	Name     string
	Codec    string
	Type     TrackType
	Language string
}

// Lang tries to return the lang of the track entry
func (t *TrackEntry) Lang() (polochon.Language, bool) {
	if t.Language != "" {
		l, ok := subtitleLangMap[t.Language]
		if ok {
			return l, true
		}
	}

	for _, prefix := range subtitlePrefixEnglish {
		if strings.HasPrefix(t.Language, prefix) {
			return polochon.EN, true
		}
	}

	return polochon.Language(""), false
}

// PrettyCodec tries to return the codec in a pretty format
func (t *TrackEntry) PrettyCodec() string {
	var out string
	var ok bool

	switch t.Type {
	case TrackTypeVideo:
		out, ok = videoCodecMap[t.Codec]
	case TrackTypeAudio:
		out, ok = audioCodecMap[t.Codec]
	default:
		return ""
	}

	if !ok {
		return ""
	}

	return out
}
