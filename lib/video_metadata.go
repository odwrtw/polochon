package polochon

import "time"

// VideoMetadata represents metadatas of the video file
type VideoMetadata struct {
	DateAdded    time.Time `json:"date_added"`
	Quality      Quality   `json:"quality"`
	ReleaseGroup string    `json:"release_group"`
	AudioCodec   string    `json:"audio_codec"`
	VideoCodec   string    `json:"video_codec"`
	Container    string    `json:"container"`
}

// Update updates the metadata with new values
func (v *VideoMetadata) Update(metadata *VideoMetadata) {
	if metadata == nil {
		return
	}

	if !metadata.DateAdded.IsZero() {
		v.DateAdded = metadata.DateAdded
	}

	if metadata.Quality != "" {
		v.Quality = metadata.Quality
	}

	for _, s := range []struct {
		o *string
		n string
	}{
		{o: &v.ReleaseGroup, n: metadata.ReleaseGroup},
		{o: &v.AudioCodec, n: metadata.AudioCodec},
		{o: &v.VideoCodec, n: metadata.VideoCodec},
		{o: &v.Container, n: metadata.Container},
	} {
		if s.n != "" {
			*s.o = s.n
		}
	}
}
