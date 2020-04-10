package transmission

import (
	"strconv"
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
)

func labels(torrent *polochon.Torrent) []string {
	if !torrent.HasVideo() {
		return nil
	}

	switch torrent.Type {
	case "movie":
		return []string{
			"type=movie",
			"imdb_id=" + torrent.ImdbID,
			"quality=" + string(torrent.Quality),
		}
	case "episode":
		return []string{
			"type=episode",
			"imdb_id=" + torrent.ImdbID,
			"quality=" + string(torrent.Quality),
			"season=" + strconv.Itoa(torrent.Season),
			"episode=" + strconv.Itoa(torrent.Episode),
		}
	default:
		return nil
	}
}

func parseLabel(label string) (string, string) {
	s := strings.Split(label, "=")
	if len(s) != 2 {
		return "", ""
	}
	return s[0], s[1]
}

func updateFromLabel(torrent *polochon.Torrent, labels []string) {
	if len(labels) == 0 {
		return
	}

	for _, label := range labels {
		k, v := parseLabel(label)
		switch k {
		case "type":
			torrent.Type = v
		case "imdb_id":
			torrent.ImdbID = v
		case "quality":
			q, err := polochon.StringToQuality(v)
			if err != nil {
				continue
			}
			torrent.Quality = *q
		case "season":
			s, err := strconv.Atoi(v)
			if err != nil {
				continue
			}
			torrent.Season = s
		case "episode":
			e, err := strconv.Atoi(v)
			if err != nil {
				continue
			}
			torrent.Episode = e
		}
	}
}
