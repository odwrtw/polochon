package polochon

import "errors"

var (
	// ErrDuplicateTorrent returned when the torrent is already added
	ErrDuplicateTorrent = errors.New("Torrent already added")
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
	ID             string   `json:"id"`
	Name           string   `json:"name"`
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

// Torrent represents a torrent file
type Torrent struct {
	ImdbID  string  `json:"imdb_id"`
	Type    string  `json:"type"`
	Season  int     `json:"season"`
	Episode int     `json:"episode"`
	Quality Quality `json:"quality"`

	Result *TorrentResult `json:"result"`
	Status *TorrentStatus `json:"status"`
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

	for _, t := range torrents {
		if t.Result == nil {
			continue
		}

		bestByQuality, ok := torrentByQuality[t.Quality]
		if !ok {
			torrentByQuality[t.Quality] = t
			continue
		}

		if t.Result.Seeders > bestByQuality.Result.Seeders {
			torrentByQuality[t.Quality] = t
		}
	}

	filtered := []*Torrent{}
	for _, t := range torrentByQuality {
		filtered = append(filtered, t)
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
