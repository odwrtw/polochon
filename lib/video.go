package polochon

// VideoType represents the type of a video
type VideoType string

// Possible video types
const (
	TypeMovie   VideoType = "movie"
	TypeEpisode VideoType = "episode"
)

// Video represents a generic video type
type Video interface {
	Subtitlable
	Detailable
	Torrentable

	SetFile(f File)
	GetFile() *File

	SetMetadata(*VideoMetadata)

	SetSubtitles([]*Subtitle)
	GetSubtitles() []*Subtitle

	SetTorrents([]*Torrent)
	GetTorrents() []*Torrent
}

// BaseVideo holds common characteristics of videos
type BaseVideo struct {
	File
	VideoMetadata
	Subtitles []*Subtitle `json:"subtitles"`
	Torrents  []*Torrent  `json:"torrents"`
}

// SetFile implements the Video interface
func (bv *BaseVideo) SetFile(f File) {
	bv.File = f
}

// GetFile implements the Video interface
func (bv *BaseVideo) GetFile() *File {
	return &bv.File
}

// SetMetadata implements the Video interface
func (bv *BaseVideo) SetMetadata(metadata *VideoMetadata) {
	bv.VideoMetadata.Update(metadata)
}

// SetSubtitles implements the Video interface
func (bv *BaseVideo) SetSubtitles(subtitles []*Subtitle) {
	bv.Subtitles = subtitles
}

// GetSubtitles implements the Video interface
func (bv *BaseVideo) GetSubtitles() []*Subtitle {
	return bv.Subtitles
}

// SetTorrents implements the Video interface
func (bv *BaseVideo) SetTorrents(torrents []*Torrent) {
	bv.Torrents = torrents
}

// GetTorrents implements the Video interface
func (bv *BaseVideo) GetTorrents() []*Torrent {
	return bv.Torrents
}
