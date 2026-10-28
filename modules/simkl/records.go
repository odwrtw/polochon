package simkl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

type number string

func (n *number) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, (*string)(n))
	}
	var v json.Number
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*n = number(v.String())
	return nil
}
func (n number) integer() int      { i, _ := strconv.Atoi(string(n)); return i }
func (n number) floating() float32 { f, _ := strconv.ParseFloat(string(n), 32); return float32(f) }

type ids struct {
	Simkl   number `json:"simkl"`
	SimklID number `json:"simkl_id"`
	IMDb    string `json:"imdb"`
	TMDB    number `json:"tmdb"`
	TVDB    number `json:"tvdb"`
	Slug    string `json:"slug"`
}

func (i ids) simkl() string {
	if i.Simkl.integer() > 0 {
		return string(i.Simkl)
	}
	if i.SimklID.integer() > 0 {
		return string(i.SimklID)
	}
	return ""
}

type rating struct {
	Rating number `json:"rating"`
	Votes  number `json:"votes"`
}
type record struct {
	Title        string            `json:"title"`
	Year         number            `json:"year"`
	Type         string            `json:"type"`
	EndpointType string            `json:"endpoint_type"`
	URL          string            `json:"url"`
	IDs          ids               `json:"ids"`
	Overview     string            `json:"overview"`
	Poster       string            `json:"poster"`
	Fanart       string            `json:"fanart"`
	Banner       string            `json:"banner"`
	Runtime      number            `json:"runtime"`
	Ratings      map[string]rating `json:"ratings"`
	Genres       []string          `json:"genres"`
	Released     string            `json:"released"`
	ReleaseDate  string            `json:"release_date"`
	FirstAired   string            `json:"first_aired"`
	Airs         struct {
		Timezone string `json:"timezone"`
	} `json:"airs"`
}
type episodeRecord struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Runtime     number `json:"runtime"`
	Season      number `json:"season"`
	Episode     number `json:"episode"`
	Type        string `json:"type"`
	Aired       *bool  `json:"aired"`
	Date        string `json:"date"`
	Image       string `json:"img"`
	IDs         ids    `json:"ids"`
}

func (e episodeRecord) validCoordinates() bool {
	season, seasonErr := strconv.Atoi(string(e.Season))
	episode, episodeErr := strconv.Atoi(string(e.Episode))
	return seasonErr == nil && episodeErr == nil && season >= 0 && episode > 0 && (e.Type == "" || e.Type == "episode" || e.Type == "special")
}

var imdbPattern = regexp.MustCompile(`^tt[0-9]+$`)
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (r record) catalogKind(kind string) bool {
	for _, v := range []string{r.Type, r.EndpointType} {
		if v != "" && !((kind == "movies" && (v == "movie" || v == "movies")) || (kind == "tv" && (v == "show" || v == "tv"))) {
			return false
		}
	}
	if r.URL != "" {
		u, err := url.Parse(r.URL)
		if err != nil {
			return false
		}
		if strings.HasPrefix(u.Path, "/anime/") {
			return false
		}
	}
	return r.IDs.simkl() != ""
}

func (r record) compatible(kind string) bool {
	return r.catalogKind(kind) && imdbPattern.MatchString(r.IDs.IMDb)
}

func imageURL(path, kind string) string {
	if path == "" {
		return ""
	}
	u, err := url.Parse(path)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		if u.Scheme == "https" || u.Scheme == "http" {
			return path
		}
		return ""
	}
	path = strings.TrimPrefix(path, "/")
	if strings.HasPrefix(path, kind+"/") {
		path = strings.TrimPrefix(path, kind+"/")
	}
	suffix := "_w.webp"
	if kind == "fanart" {
		suffix = "_d.jpg"
	}
	if strings.Contains(path, "?") || strings.Contains(path, "..") {
		return ""
	}
	if strings.HasSuffix(path, ".webp") || strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".png") {
		suffix = ""
	}
	return "https://simkl.in/" + kind + "/" + path + suffix
}

func (r record) link(kind string) string {
	return "https://simkl.com/" + kind + "/" + r.IDs.simkl() + "/" + url.PathEscape(r.IDs.Slug)
}
func (r record) year() int {
	if r.Year.integer() > 0 {
		return r.Year.integer()
	}
	for _, d := range []string{r.Released, r.FirstAired, r.ReleaseDate} {
		for _, layout := range []string{time.RFC3339, "2006-01-02", "01/02/2006"} {
			if t, err := time.Parse(layout, d); err == nil {
				return t.Year()
			}
		}
	}
	return 0
}
func runtimeMinutes(n number) int {
	if n.integer() > 0 {
		return n.integer()
	}
	var h, m int
	s := string(n)
	if strings.Contains(s, "h") {
		parts := strings.SplitN(s, "h", 2)
		h, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
		s = parts[1]
	}
	m, _ = strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "m")))
	return h*60 + m
}

func (r record) movie(target *polochon.Movie) {
	if r.IDs.IMDb != "" {
		target.ImdbID = r.IDs.IMDb
	}
	if r.Title != "" {
		target.Title = r.Title
		target.OriginalTitle = r.Title
		target.SortTitle = r.Title
	}
	if r.Overview != "" {
		target.Plot = r.Overview
	}
	if y := r.year(); y > 0 {
		target.Year = y
	}
	if r.IDs.TMDB.integer() > 0 {
		target.TmdbID = r.IDs.TMDB.integer()
	}
	if r.Runtime != "" {
		target.Runtime = runtimeMinutes(r.Runtime)
	}
	if len(r.Genres) > 0 {
		target.Genres = append([]string(nil), r.Genres...)
	}
	if v, ok := r.Ratings["simkl"]; ok {
		target.Rating = v.Rating.floating()
		target.Votes = v.Votes.integer()
	}
	if v := imageURL(r.Poster, "posters"); v != "" {
		target.ThumbURL = v
	}
	if v := imageURL(r.Fanart, "fanart"); v != "" {
		target.FanartURL = v
	}
}
func (r record) show(target *polochon.Show) {
	if r.IDs.IMDb != "" {
		target.ImdbID = r.IDs.IMDb
	}
	if r.Title != "" {
		target.Title = r.Title
	}
	if r.Overview != "" {
		target.Plot = r.Overview
	}
	if y := r.year(); y > 0 {
		target.Year = y
	}
	if r.IDs.TVDB.integer() > 0 {
		target.TvdbID = r.IDs.TVDB.integer()
	}
	if v, ok := r.Ratings["simkl"]; ok {
		target.Rating = v.Rating.floating()
	}
	if v := imageURL(r.Poster, "posters"); v != "" {
		target.PosterURL = v
	}
	if v := imageURL(r.Fanart, "fanart"); v != "" {
		target.FanartURL = v
	}
	// Simkl has no documented native banner family. Only use an actual supplied URL.
	if u, err := url.Parse(r.Banner); err == nil && u.IsAbs() && (u.Scheme == "http" || u.Scheme == "https") {
		target.BannerURL = r.Banner
	}
	if t, err := time.Parse(time.RFC3339, r.FirstAired); err == nil {
		target.FirstAired = &t
	}
	if r.IDs.simkl() != "" {
		target.URL = r.link("tv")
	}
}
func (e episodeRecord) episode(show *polochon.Show) *polochon.ShowEpisode {
	v := polochon.NewShowEpisode(show.ShowConfig)
	v.Title = e.Title
	v.Plot = e.Description
	v.Runtime = runtimeMinutes(e.Runtime)
	v.Season = e.Season.integer()
	v.Episode = e.Episode.integer()
	v.Aired = e.Date
	v.Show = show
	v.ShowTitle = show.Title
	v.ShowImdbID = show.ImdbID
	v.ShowTvdbID = show.TvdbID
	v.EpisodeImdbID = e.IDs.IMDb
	v.TvdbID = e.IDs.TVDB.integer()
	v.ThumbURL = imageURL(e.Image, "episodes")
	return v
}

func (c *client) remember(kind string, r record) {
	if !r.compatible(kind) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for field, value := range map[string]string{"imdb": r.IDs.IMDb, "tmdb": string(r.IDs.TMDB), "tvdb": string(r.IDs.TVDB)} {
		if value == "" {
			continue
		}
		key := kind + ":" + field + ":" + value
		if c.mappings[key] == nil {
			if len(c.mappings) >= 4096 {
				for k := range c.mappings {
					delete(c.mappings, k)
					break
				}
			}
			c.mappings[key] = make(map[string]struct{})
		}
		if len(c.mappings[key]) < 2 {
			c.mappings[key][r.IDs.simkl()] = struct{}{}
		}
	}
}

func (c *client) known(kind, field, value string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	set := c.mappings[kind+":"+field+":"+value]
	if len(set) > 1 {
		return "", ErrAmbiguous
	}
	for id := range set {
		return id, nil
	}
	return "", nil
}

func episodeKey(e episodeRecord) string {
	return fmt.Sprintf("%d:%d", e.Season.integer(), e.Episode.integer())
}
