package simkl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

type genre struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Simkl) genres(ctx context.Context, kind string) ([]genre, error) {
	c, err := s.ready()
	if err != nil {
		return nil, err
	}
	var list []genre
	err = c.get(ctx, c.cdn+"/discover/trending/"+kind+"/genre/index.json", false, 24*time.Hour, true, &list)
	return list, err
}

func (s *Simkl) options(kind string) []string {
	options := []string{"simkl-trending-today", "simkl-trending-week", "simkl-trending-month", "simkl-trending-today-500", "simkl-trending-week-500", "simkl-trending-month-500"}
	if kind == "movies" {
		options = append(options, "simkl-dvd", "simkl-dvd-500")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	genres, err := s.genres(ctx, kind)
	if err != nil {
		s.log.Warn("genre options unavailable; returning built-in lists", "error", err)
		return options
	}
	for _, g := range genres {
		if slugPattern.MatchString(g.Slug) {
			options = append(options, "simkl-genre:"+g.Slug)
		}
	}
	return options
}

func (s *Simkl) AvailableMovieOptions() []string { return s.options("movies") }
func (s *Simkl) AvailableShowOptions() []string  { return s.options("tv") }

func (s *Simkl) list(ctx context.Context, kind, option string) ([]record, error) {
	c, err := s.ready()
	if err != nil {
		return nil, err
	}
	var path string
	ttl := 24 * time.Hour
	if strings.HasPrefix(option, "simkl-trending-") {
		period := strings.TrimPrefix(option, "simkl-trending-")
		size := "100"
		if strings.HasSuffix(period, "-500") {
			size = "500"
			period = strings.TrimSuffix(period, "-500")
		}
		if period != "today" && period != "week" && period != "month" {
			return nil, ErrUnsupported
		}
		if period == "today" {
			ttl = time.Hour
		}
		path = "/discover/trending/" + kind + "/" + period + "_" + size + ".json"
	} else if kind == "movies" && (option == "simkl-dvd" || option == "simkl-dvd-500") {
		size := "100"
		if option == "simkl-dvd-500" {
			size = "500"
		}
		path = "/discover/dvd/releases_" + size + ".json"
	} else if slug, ok := strings.CutPrefix(option, "simkl-genre:"); ok && slugPattern.MatchString(slug) {
		genres, err := s.genres(ctx, kind)
		if err != nil {
			return nil, err
		}
		for _, g := range genres {
			if g.Slug == slug {
				path = "/discover/trending/" + kind + "/genre/" + slug + "_month_500.json"
				break
			}
		}
		if path == "" {
			return nil, ErrUnsupported
		}
	} else {
		return nil, ErrUnsupported
	}
	var records []record
	if err = c.get(ctx, c.cdn+path, false, ttl, true, &records); err != nil {
		return nil, err
	}
	if len(records) > 500 {
		records = records[:500]
	}
	return s.addressable(kind, records), nil
}

func (s *Simkl) addressable(kind string, records []record) []record {
	out := make([]record, 0, len(records))
	skipped := 0
	seen := make(map[string]bool)
	identities := make(map[string]string)
	conflicts := make(map[string]bool)
	for _, r := range records {
		if !r.compatible(kind) {
			skipped++
			continue
		}
		if previous := identities[r.IDs.IMDb]; previous != "" && previous != r.IDs.simkl() {
			conflicts[r.IDs.IMDb] = true
		}
		identities[r.IDs.IMDb] = r.IDs.simkl()
		s.client.remember(kind, r)
	}
	for _, r := range records {
		if !r.compatible(kind) {
			continue
		}
		if _, err := s.client.known(kind, "imdb", r.IDs.IMDb); err != nil || conflicts[r.IDs.IMDb] {
			skipped++
			continue
		}
		if seen[r.IDs.IMDb] {
			continue
		}
		seen[r.IDs.IMDb] = true
		out = append(out, r)
	}
	if skipped > 0 {
		s.log.Warn("skipped unsupported catalog identities", "kind", kind, "count", skipped)
	}
	return out
}

func (s *Simkl) GetMovieList(ctx context.Context, option string) ([]*polochon.Movie, error) {
	records, err := s.list(ctx, "movies", option)
	if err != nil {
		return nil, err
	}
	list := make([]*polochon.Movie, 0, len(records))
	for _, r := range records {
		m := polochon.NewMovie(polochon.MovieConfig{})
		r.movie(m)
		list = append(list, m)
	}
	return list, nil
}
func (s *Simkl) GetShowList(ctx context.Context, option string) ([]*polochon.Show, error) {
	records, err := s.list(ctx, "tv", option)
	if err != nil {
		return nil, err
	}
	list := make([]*polochon.Show, 0, len(records))
	for _, r := range records {
		show := polochon.NewShow(polochon.ShowConfig{})
		r.show(show)
		list = append(list, show)
	}
	return list, nil
}

func (s *Simkl) search(ctx context.Context, kind, key string) ([]record, error) {
	c, err := s.ready()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(key) == "" {
		return nil, ErrUnsupported
	}
	pathKind := kind
	if kind == "movies" {
		pathKind = "movie"
	}
	q := url.Values{"q": {key}, "limit": {"20"}, "page": {"1"}, "extended": {"full"}}
	var records []record
	if err = c.get(ctx, c.api+"/search/"+pathKind+"?"+q.Encode(), true, 5*time.Minute, false, &records); err != nil {
		return nil, err
	}
	if len(records) > 20 {
		records = records[:20]
	}
	// The documented text-search ID block contains Simkl/TMDB IDs, but no IMDb.
	// Resolve only these bounded matches through public, cached details so they
	// can be safely represented by the existing Polochon search result types.
	for i, candidate := range records {
		if !candidate.catalogKind(kind) || imdbPattern.MatchString(candidate.IDs.IMDb) {
			continue
		}
		full, err := s.details(ctx, kind, candidate.IDs.simkl())
		if err != nil {
			if errors.Is(err, ErrUnsupported) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrAmbiguous) {
				continue
			}
			return nil, err
		}
		records[i] = full
	}
	return s.addressable(kind, records), nil
}
func (s *Simkl) SearchMovie(ctx context.Context, key string) ([]*polochon.Movie, error) {
	records, err := s.search(ctx, "movies", key)
	if err != nil {
		return nil, err
	}
	out := make([]*polochon.Movie, 0, len(records))
	for _, r := range records {
		m := polochon.NewMovie(polochon.MovieConfig{})
		r.movie(m)
		out = append(out, m)
	}
	return out, nil
}
func (s *Simkl) SearchShow(ctx context.Context, key string) ([]*polochon.Show, error) {
	records, err := s.search(ctx, "tv", key)
	if err != nil {
		return nil, err
	}
	out := make([]*polochon.Show, 0, len(records))
	for _, r := range records {
		show := polochon.NewShow(polochon.ShowConfig{})
		r.show(show)
		out = append(out, show)
	}
	return out, nil
}

func (s *Simkl) resolve(ctx context.Context, kind, imdb string, numeric int, title string, year int) (string, error) {
	c, err := s.ready()
	if err != nil {
		return "", err
	}
	q := url.Values{}
	if imdb != "" {
		if !imdbPattern.MatchString(imdb) {
			return "", ErrUnsupported
		}
		q.Set("imdb", imdb)
		if id, err := c.known(kind, "imdb", imdb); err != nil || id != "" {
			return id, err
		}
	} else if numeric > 0 {
		field := "tvdb"
		if kind == "movies" {
			field = "tmdb"
		}
		value := fmt.Sprint(numeric)
		q.Set(field, value)
		if id, err := c.known(kind, field, value); err != nil || id != "" {
			return id, err
		}
	} else {
		if title == "" {
			return "", ErrUnsupported
		}
		records, err := s.search(ctx, kind, title)
		if err != nil {
			return "", err
		}
		found := ""
		for _, r := range records {
			if strings.EqualFold(r.Title, title) && (year == 0 || r.year() == year) {
				if found != "" && found != r.IDs.simkl() {
					return "", ErrAmbiguous
				}
				found = r.IDs.simkl()
			}
		}
		if found == "" {
			return "", ErrNotFound
		}
		return found, nil
	}
	// External-ID resolution is an authenticated operation even for public records.
	typeName := "show"
	if kind == "movies" {
		typeName = "movie"
	}
	q.Set("type", typeName)
	var records []record
	if err = c.get(ctx, c.api+"/search/id?"+q.Encode(), true, 5*time.Minute, false, &records); err != nil {
		return "", err
	}
	var found string
	for _, r := range records {
		if r.Type != "" && !((kind == "movies" && r.Type == "movie") || (kind == "tv" && (r.Type == "tv" || r.Type == "show"))) {
			continue
		}
		if r.IDs.simkl() == "" {
			continue
		}
		if found != "" && found != r.IDs.simkl() {
			return "", ErrAmbiguous
		}
		found = r.IDs.simkl()
	}
	if found == "" {
		return "", ErrNotFound
	}
	return found, nil
}
