package polochon

import "errors"

var (
	// ErrDuplicateTorrent returned when the torrent is already added
	ErrDuplicateTorrent = errors.New("Torrent already added")
)

// TorrentMetadata represent the metadata of a torrent
type TorrentMetadata struct {
	ImdbID  string  `json:"imdb_id"`
	Type    string  `json:"type"`
	Season  int     `json:"season"`
	Episode int     `json:"episode"`
	Quality Quality `json:"quality"`
}

// Torrent represents a torrent file
type Torrent struct {
	// Generic properties
	Name       string           `json:"name"`
	Quality    Quality          `json:"quality"`
	URL        string           `json:"url"`
	Seeders    int              `json:"seeders"`
	Leechers   int              `json:"leechers"`
	Source     string           `json:"source"`
	UploadUser string           `json:"upload_user"`
	Size       int              `json:"size"`
	Metadata   *TorrentMetadata `json:"metadata"`

	// Properties once downloading
	ID             string   `json:"id"`
	Ratio          float32  `json:"ratio"`
	IsFinished     bool     `json:"is_finished"`
	FilePaths      []string `json:"file_paths"`
	DownloadRate   int      `json:"download_rate"`
	UploadRate     int      `json:"upload_rate"`
	TotalSize      int      `json:"total_size"`
	DownloadedSize int      `json:"downloaded_size"`
	UploadedSize   int      `json:"uploaded_size"`
	PercentDone    float32  `json:"percent_done"`
}

// RatioReached tells if the given ratio has been reached
func (t *Torrent) RatioReached(ratio float32) bool {
	if !t.IsFinished {
		return false
	}

	return t.Ratio >= ratio
}

// FilterTorrents filters the torrents to keep only the best ones
func FilterTorrents(torrents []*Torrent) []*Torrent {
	torrentByQuality := map[Quality]*Torrent{}

	for _, t := range torrents {
		bestByQuality, ok := torrentByQuality[t.Quality]
		if !ok {
			torrentByQuality[t.Quality] = t
			continue
		}

		if t.Seeders > bestByQuality.Seeders {
			torrentByQuality[t.Quality] = t
		}
	}

	filtered := []*Torrent{}
	for _, t := range torrentByQuality {
		filtered = append(filtered, t)
	}

	return filtered
}
