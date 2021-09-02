package mkvinfo

import polochon "github.com/odwrtw/polochon/lib"

// HasSubtitle tries to find the subtitle in the tracks
func HasSubtitle(tracks []*TrackEntry, lang polochon.Language) bool {
	for _, track := range tracks {
		if track.Type != TrackTypeSubtitle {
			continue
		}

		found, ok := track.Lang()
		if !ok {
			continue
		}

		if found != lang {
			continue
		}

		return true
	}

	return false
}

// Metadata returns metadata from the track entries
func Metadata(tracks []*TrackEntry) *polochon.VideoMetadata {
	m := &polochon.VideoMetadata{
		AudioCodec: "",
		VideoCodec: "",
		Container:  "mkv",
	}

	subtitles := map[polochon.Language]struct{}{}

	for _, track := range tracks {
		switch track.Type {
		case TrackTypeSubtitle:
			lang, ok := track.Lang()
			if ok {
				subtitles[lang] = struct{}{}
			}
		case TrackTypeVideo:
			m.VideoCodec = track.PrettyCodec()
		case TrackTypeAudio:
			m.AudioCodec = track.PrettyCodec()
		}
	}

	c := len(subtitles)
	if c > 0 {
		m.EmbeddedSubtitles = make([]polochon.Language, c)
		i := 0
		for lang := range subtitles {
			m.EmbeddedSubtitles[i] = lang
			i++
		}
	}

	if m.VideoCodec == "" && m.AudioCodec == "" && len(m.EmbeddedSubtitles) == 0 {
		return nil
	}

	return m
}
