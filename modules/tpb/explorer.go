package tpb

import (
	"context"
	"errors"
	"time"

	"github.com/odwrtw/whatsthis"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/tpb"
	"github.com/sirupsen/logrus"
)

var (
	// ErrInvalidArgument returned when an invalid object is passed
	ErrInvalidArgument = errors.New("tpb: invalid argument")
)

// AvailableMovieOptions implements the Explorer interface
func (t *TPB) AvailableMovieOptions() []string {
	return []string{
		"top100",
		"top100 HD",
	}
}

// GetMovieList implements the Explorer interface
func (t *TPB) GetMovieList(option string, log *logrus.Entry) ([]*polochon.Movie, error) {
	var category tpb.TorrentCategory
	switch option {
	case "top100":
		category = tpb.VideoMovies
	case "top100 HD":
		category = tpb.VideoHDMovies
	default:
		return nil, ErrInvalidArgument
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	torrents, err := t.Client.Top100(ctx, category)
	if err != nil {
		return nil, err
	}

	result := []*polochon.Movie{}
	for _, torrent := range torrents {
		torrentStr := torrentGuessitStr(torrent)
		guess := whatsthis.Video(torrentStr)

		// Create the movie
		m := polochon.NewMovie(polochon.MovieConfig{})
		m.Title = guess.Title
		m.Year = guess.Year
		m.ImdbID = torrent.ImdbID

		// Set the default quality if none is defined
		screenSize := guess.ScreenSize
		if screenSize == "" {
			screenSize = string(polochon.Quality720p)
		}

		torrentQuality := polochon.Quality(screenSize)
		if !torrentQuality.IsAllowed() {
			log.Debugf("tpb: unhandled quality: %q", torrentQuality)
			continue
		}
		m.Torrents = append(m.Torrents, &polochon.Torrent{
			Quality: torrentQuality,
			Result: &polochon.TorrentResult{
				URL:      torrent.Magnet(),
				Seeders:  torrent.Seeders,
				Leechers: torrent.Leechers,
				Source:   moduleName,
			},
		})

		// Append the movie
		result = append(result, m)
	}
	return result, nil
}

// AvailableShowOptions implements the Explorer interface
func (t *TPB) AvailableShowOptions() []string {
	return []string{
		"top100",
		"top100 HD",
	}
}

// GetShowList implements the Explorer interface
func (t *TPB) GetShowList(option string, log *logrus.Entry) ([]*polochon.Show, error) {
	var category tpb.TorrentCategory
	switch option {
	case "top100":
		category = tpb.VideoTVshows
	case "top100 HD":
		category = tpb.VideoHDTVshows
	default:
		return nil, ErrInvalidArgument
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	torrents, err := t.Client.Top100(ctx, category)
	if err != nil {
		return nil, err
	}

	result := []*polochon.Show{}
	for _, torrent := range torrents {
		torrentStr := torrentGuessitStr(torrent)
		guess := whatsthis.Video(torrentStr)

		// Create the show
		s := polochon.NewShow(polochon.ShowConfig{})
		s.Title = guess.Title
		s.Year = guess.Year
		s.ImdbID = torrent.ImdbID

		// Append the show
		result = append(result, s)
	}
	return result, nil
}
