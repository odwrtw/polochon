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

	for _, track := range tracks {
		if track.Type == TrackTypeSubtitle {
			continue
		}

		if track.Type == TrackTypeVideo {
			m.VideoCodec = track.PrettyCodec()
		}

		if track.Type == TrackTypeAudio {
			m.AudioCodec = track.PrettyCodec()
		}

		if m.VideoCodec != "" && m.AudioCodec != "" {
			break
		}
	}

	if m.VideoCodec == "" && m.AudioCodec == "" {
		return nil
	}

	return m
}
