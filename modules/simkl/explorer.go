package simkl

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

var _ polochon.Explorer = (*Simkl)(nil)

func (s *Simkl) AvailableMovieOptions() []string {
	return []string{
		"trending", "trending-week", "trending-month", "dvd",
		"popular-this-week", "popular-this-month", "popular-all-time",
		"rank", "release-date", "voted", "watched",
	}
}

func (s *Simkl) AvailableShowOptions() []string {
	return []string{
		"trending", "trending-week", "trending-month",
		"popular-this-week", "popular-this-month", "popular-all-time",
		"rank", "release-date", "voted", "watched",
		"best", "best-year", "best-month", "best-voted", "best-watched",
		"premieres", "upcoming", "airing", "airing-tomorrow",
	}
}

func (s *Simkl) GetMovieList(ctx context.Context, option string) ([]*polochon.Movie, error) {
	items, err := s.explore(ctx, "movies", option)
	if err != nil {
		return nil, err
	}
	return moviesFromListings(items), nil
}

func (s *Simkl) GetShowList(ctx context.Context, option string) ([]*polochon.Show, error) {
	items, err := s.explore(ctx, "tv", option)
	if err != nil {
		return nil, err
	}
	return showsFromListings(items), nil
}

func (s *Simkl) explore(ctx context.Context, kind, option string) ([]listing, error) {
	path, params, feed, err := explorerRequest(kind, option)
	if err != nil {
		return nil, err
	}
	if s.client == nil {
		return nil, ErrMissingClientID
	}
	var items []listing
	if feed {
		var entries []feedItem
		if err := s.client.get(ctx, s.client.dataURL, path, nil, false, &entries); err != nil {
			return nil, err
		}
		items = make([]listing, 0, min(40, len(entries)))
		for _, entry := range entries[:min(40, len(entries))] {
			if entry.Year == 0 && entry.ReleaseDate != "" {
				date, err := time.Parse("01/02/2006", entry.ReleaseDate)
				if err != nil {
					return nil, ErrInvalidResponse
				}
				entry.Year = date.Year()
			}
			items = append(items, entry.listing)
		}
	} else {
		if err := s.client.get(ctx, s.client.baseURL, path, params, true, &items); err != nil {
			return nil, err
		}
		items = items[:min(40, len(items))]
	}
	return s.usableListings(ctx, kind, items)
}

func explorerRequest(kind, option string) (string, url.Values, bool, error) {
	params := url.Values{}
	switch option {
	case "trending", "trending-week", "trending-month":
		period := strings.TrimPrefix(option, "trending-")
		if option == "trending" {
			period = "today"
		}
		return "/discover/trending/" + kind + "/" + period + "_100.json", nil, true, nil
	case "dvd":
		if kind == "movies" {
			return "/discover/dvd/releases_100.json", nil, true, nil
		}
	case "popular-this-week", "popular-this-month", "popular-all-time", "rank", "release-date", "voted", "watched":
		path := "/movies/genres/all/movies/all/all/" + option
		if kind == "tv" {
			path = "/tv/genres/all/all/all/all/all/" + option
		}
		params.Set("page", "1")
		params.Set("limit", "40")
		return path, params, false, nil
	case "best", "best-year", "best-month", "best-voted", "best-watched":
		if kind == "tv" {
			filter := strings.TrimPrefix(option, "best-")
			if option == "best" {
				filter = "all"
			}
			return "/tv/best/" + filter, params, false, nil
		}
	case "premieres", "upcoming":
		if kind == "tv" {
			filter := "new"
			if option == "upcoming" {
				filter = "soon"
			}
			params.Set("page", "1")
			params.Set("limit", "40")
			return "/tv/premieres/" + filter, params, false, nil
		}
	case "airing", "airing-tomorrow":
		if kind == "tv" {
			date := "today"
			if option == "airing-tomorrow" {
				date = "tomorrow"
			}
			params.Set("date", date)
			return "/tv/airing", params, false, nil
		}
	}
	return "", nil, false, ErrInvalidArgument
}

// Most API lists lack external IDs; hydrate only when IMDb identity is missing.
// Trending feeds normally already contain it. Never return unusable library keys.
func (s *Simkl) usableListings(ctx context.Context, kind string, items []listing) ([]listing, error) {
	result := make([]listing, 0, len(items))
	seenIDs := map[int]bool{}
	seenImdb := map[string]bool{}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		id := item.IDs.id()
		if id <= 0 || item.Title == "" || seenIDs[id] {
			continue
		}
		seenIDs[id] = true
		if item.IDs.Imdb == "" {
			details, err := s.client.detail(ctx, kind, id)
			if errors.Is(err, ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
			item = details.listing
		}
		if item.IDs.Imdb == "" || seenImdb[item.IDs.Imdb] {
			continue
		}
		seenImdb[item.IDs.Imdb] = true
		result = append(result, item)
	}
	return result, nil
}

func moviesFromListings(items []listing) []*polochon.Movie {
	movies := make([]*polochon.Movie, 0, len(items))
	for _, item := range items {
		movie := polochon.NewMovie(polochon.MovieConfig{})
		item.updateMovie(movie)
		movies = append(movies, movie)
	}
	return movies
}

func showsFromListings(items []listing) []*polochon.Show {
	shows := make([]*polochon.Show, 0, len(items))
	for _, item := range items {
		show := polochon.NewShow(polochon.ShowConfig{})
		item.updateShow(show)
		shows = append(shows, show)
	}
	return shows
}
