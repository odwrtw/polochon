package polochon

import (
	"errors"
	"sort"
)

var (
	// ErrDuplicateTorrent returned when the torrent is already added
	ErrDuplicateTorrent = errors.New("Torrent already added")
)

// TorrentState represents the torrent state
type TorrentState string

// Possible torrent statuses
const (
	TorrentStateStopped         TorrentState = "stopped"
	TorrentStatePending         TorrentState = "pending"
	TorrentStateChecking        TorrentState = "checking"
	TorrentStateDownloadPending TorrentState = "download_pending"
	TorrentStateDownloading     TorrentState = "downloading"
	TorrentStateSeedPending     TorrentState = "seed_pending"
	TorrentStateSeeding         TorrentState = "seeding"
)

// TorrentResult represents a torrent result from a search
type TorrentResult struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Seeders    int    `json:"seeders"`
	Leechers   int    `json:"leechers"`
	Source     string `json:"source"`
	UploadUser string `json:"upload_user"`
	Size       int    `json:"size"`
}

// TorrentStatus represents the status of the downloaded torrent
type TorrentStatus struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Ratio          float32      `json:"ratio"`
	IsFinished     bool         `json:"is_finished"`
	FilePaths      []string     `json:"file_paths"`
	DownloadRate   int          `json:"download_rate"`
	UploadRate     int          `json:"upload_rate"`
	TotalSize      int          `json:"total_size"`
	DownloadedSize int          `json:"downloaded_size"`
	UploadedSize   int          `json:"uploaded_size"`
	PercentDone    float32      `json:"percent_done"`
	State          TorrentState `json:"state"`
}

// Torrent represents a torrent file
type Torrent struct {
	ImdbID  string    `json:"imdb_id"`
	Type    VideoType `json:"type"`
	Season  int       `json:"season"`
	Episode int       `json:"episode"`
	Quality Quality   `json:"quality"`

	Result *TorrentResult `json:"result"`
	Status *TorrentStatus `json:"status"`
}

// HasVideo returns true if the torrent has enough information to return a
// video
func (t *Torrent) HasVideo() bool {
	if t.ImdbID == "" || string(t.Type) == "" {
		return false
	}

	if t.Type == TypeMovie {
		return true
	}

	if t.Type != TypeEpisode {
		return false
	}

	return (t.Season != 0 && t.Episode != 0)
}

// Video returns a new video based on the torrent informations
func (t *Torrent) Video() Video {
	if !t.HasVideo() {
		return nil
	}

	var video Video

	switch t.Type {
	case TypeMovie:
		video = &Movie{
			ImdbID: t.ImdbID,
		}
	case TypeEpisode:
		show := &Show{ImdbID: t.ImdbID}
		episode := &ShowEpisode{
			ShowImdbID: t.ImdbID,
			Season:     t.Season,
			Episode:    t.Episode,
			Show:       show,
		}
		show.Episodes = []*ShowEpisode{episode}
		video = episode
	default:
		return nil
	}

	video.SetTorrents([]*Torrent{t})
	video.SetMetadata(&VideoMetadata{
		Quality: t.Quality,
	})

	return video
}

// RatioReached tells if the given ratio has been reached
func (t *Torrent) RatioReached(ratio float32) bool {
	if t.Status == nil || !t.Status.IsFinished {
		return false
	}

	return t.Status.Ratio >= ratio
}

// FilterTorrents filters the torrents to keep only the best ones
func FilterTorrents(torrents []*Torrent) []*Torrent {
	torrentByQuality := map[Quality]*Torrent{}

	// Keep the qualities in an array to produce a predictable output
	qualities := []string{}

	for _, t := range torrents {
		if t.Result == nil {
			continue
		}

		bestByQuality, ok := torrentByQuality[t.Quality]
		if !ok {
			torrentByQuality[t.Quality] = t
			qualities = append(qualities, string(t.Quality))
			continue
		}

		if t.Result.Seeders > bestByQuality.Result.Seeders {
			torrentByQuality[t.Quality] = t
		}
	}

	sort.Strings(qualities)
	filtered := make([]*Torrent, len(qualities))
	for i, q := range qualities {
		filtered[i] = torrentByQuality[Quality(q)]
	}

	return filtered
}

// ChooseTorrentFromQualities chooses the best torrent matching the given
// qualities
func ChooseTorrentFromQualities(torrents []*Torrent, qualities []Quality) *Torrent {
	for _, quality := range qualities {
		for _, torrent := range torrents {
			if torrent.Quality == quality {
				return torrent
			}
		}
	}

	return nil
}
