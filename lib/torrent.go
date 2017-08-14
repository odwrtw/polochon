package polochon

// Torrent represents a torrent file
type Torrent struct {
	Name       string  `json:"name"`
	Quality    Quality `json:"quality"`
	URL        string  `json:"url"`
	Seeders    int     `json:"seeders"`
	Leechers   int     `json:"leechers"`
	Source     string  `json:"source"`
	UploadUser string  `json:"upload_user"`
}

// FilterTorrents filters the torrents to keep only the best ones
func FilterTorrents(torrents []Torrent) []Torrent {
	torrentByQuality := map[Quality]Torrent{}

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

	filtered := []Torrent{}
	for _, t := range torrentByQuality {
		filtered = append(filtered, t)
	}

	return filtered
}
