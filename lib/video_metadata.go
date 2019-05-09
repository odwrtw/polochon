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
