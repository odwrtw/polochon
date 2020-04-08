package transmission

import (
	"strconv"
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
)

func isValidMetadata(metadata *polochon.DownloadableMetadata) bool {
	if metadata == nil {
		return false
	}

	if metadata.ImdbID == "" || metadata.Type == "" || metadata.Quality == "" {
		return false
	}

	switch metadata.Type {
	case "movie":
		return true
	case "episode":
		if metadata.Season == 0 || metadata.Episode == 0 {
			return false
		}
		return true
	default:
		return false
	}
}

func labels(metadata *polochon.DownloadableMetadata) []string {
	if !isValidMetadata(metadata) {
		return nil
	}

	if metadata.Type == "movie" {
		return []string{
			"type=movie",
			"imdb_id=" + metadata.ImdbID,
			"quality=" + string(metadata.Quality),
		}
	}

	return []string{
		"type=episode",
		"imdb_id=" + metadata.ImdbID,
		"quality=" + string(metadata.Quality),
		"season=" + strconv.Itoa(metadata.Season),
		"episode=" + strconv.Itoa(metadata.Episode),
	}
}

func parseLabel(label string) (string, string) {
	s := strings.Split(label, "=")
	if len(s) != 2 {
		return "", ""
	}
	return s[0], s[1]
}

func metadata(labels []string) *polochon.DownloadableMetadata {
	if len(labels) == 0 {
		return nil
	}

	m := &polochon.DownloadableMetadata{}
	for _, label := range labels {
		k, v := parseLabel(label)
		switch k {
		case "type":
			m.Type = v
		case "imdb_id":
			m.ImdbID = v
		case "quality":
			q, err := polochon.StringToQuality(v)
			if err != nil {
				continue
			}
			m.Quality = *q
		case "season":
			s, err := strconv.Atoi(v)
			if err != nil {
				continue
			}
			m.Season = s
		case "episode":
			e, err := strconv.Atoi(v)
			if err != nil {
				continue
			}
			m.Episode = e
		}
	}

	if !isValidMetadata(m) {
		return nil
	}

	return m
}
