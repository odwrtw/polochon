package simkl

import (
	"net/url"
	"strconv"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

type mediaIDs struct {
	Simkl   int    `json:"simkl"`
	SimklID int    `json:"simkl_id"`
	Slug    string `json:"slug"`
	Imdb    string `json:"imdb"`
	Tmdb    string `json:"tmdb"`
	Tvdb    string `json:"tvdb"`
}

func (ids mediaIDs) id() int {
	if ids.Simkl != 0 {
		return ids.Simkl
	}
	return ids.SimklID
}

type rating struct {
	Rating float32 `json:"rating"`
	Votes  int     `json:"votes"`
}

type listing struct {
	Title        string            `json:"title"`
	Year         int               `json:"year"`
	Type         string            `json:"type"`
	EndpointType string            `json:"endpoint_type"`
	IDs          mediaIDs          `json:"ids"`
	Poster       string            `json:"poster"`
	Fanart       string            `json:"fanart"`
	Overview     string            `json:"overview"`
	Ratings      map[string]rating `json:"ratings"`
}

type feedItem struct {
	listing
	ReleaseDate string `json:"release_date"` // MM/DD/YYYY, unlike API dates
}

type mediaDetail struct {
	listing
	Runtime    int      `json:"runtime"` // feeds have a display string instead
	Genres     []string `json:"genres"`
	FirstAired string   `json:"first_aired"`
	Released   string   `json:"released"`
}

func (item listing) rating() (rating, bool) {
	if r, ok := item.Ratings["simkl"]; ok {
		return r, true
	}
	r, ok := item.Ratings["imdb"]
	return r, ok
}

func (item listing) updateMovie(movie *polochon.Movie) {
	movie.Title = item.Title
	movie.SortTitle = item.Title
	if item.IDs.Imdb != "" {
		movie.ImdbID = item.IDs.Imdb
	}
	if id, _ := strconv.Atoi(item.IDs.Tmdb); id > 0 {
		movie.TmdbID = id
	}
	if item.Year != 0 {
		movie.Year = item.Year
	}
	if item.Overview != "" {
		movie.Plot = item.Overview
	}
	if r, ok := item.rating(); ok {
		movie.Rating, movie.Votes = r.Rating, r.Votes
	}
	if item.Poster != "" {
		movie.ThumbURL = imageURL("posters", item.Poster, "_m")
	}
	if item.Fanart != "" {
		movie.FanartURL = imageURL("fanart", item.Fanart, "_medium")
	}
}

func (item listing) updateShow(show *polochon.Show) {
	show.Title = item.Title
	show.URL = "https://simkl.com/tv/" + strconv.Itoa(item.IDs.id()) + "/" + url.PathEscape(item.IDs.Slug)
	if item.IDs.Imdb != "" {
		show.ImdbID = item.IDs.Imdb
	}
	if id, _ := strconv.Atoi(item.IDs.Tvdb); id > 0 {
		show.TvdbID = id
	}
	if item.Year != 0 {
		show.Year = item.Year
	}
	if item.Overview != "" {
		show.Plot = item.Overview
	}
	if r, ok := item.rating(); ok {
		show.Rating = r.Rating
	}
	if item.Poster != "" {
		show.PosterURL = imageURL("posters", item.Poster, "_m")
		// Simkl has no dedicated banner; _w is its documented landscape crop.
		show.BannerURL = imageURL("posters", item.Poster, "_w")
	}
	if item.Fanart != "" {
		show.FanartURL = imageURL("fanart", item.Fanart, "_medium")
	}
}

func imageURL(category, path, size string) string {
	if path == "" {
		return ""
	}
	// Polochon writes these responses directly to .jpg sidecars.
	params := url.Values{
		"url": {"https://simkl.in/" + category + "/" + path + size + ".webp"},
		"q":   {"90"}, "output": {"jpg"},
	}
	return "https://wsrv.nl/?" + params.Encode()
}

func parseDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	// Simkl's date conventions and detail examples include both representations.
	for _, layout := range []string{time.RFC3339, time.DateOnly} {
		if date, err := time.Parse(layout, value); err == nil {
			return &date, nil
		}
	}
	return nil, ErrInvalidResponse
}
