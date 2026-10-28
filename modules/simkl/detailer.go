package simkl

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

var _ polochon.Detailer = (*Simkl)(nil)

type episode struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Season      int    `json:"season"`
	Episode     int    `json:"episode"`
	Description string `json:"description"`
	Runtime     int    `json:"runtime"`
	Date        string `json:"date"`
	Aired       bool   `json:"aired"`
	Image       string `json:"img"`
	// ids.simkl_id is deliberately not mapped to an external episode ID.
}

func (s *Simkl) GetDetails(ctx context.Context, value any) error {
	if s.client == nil {
		return ErrMissingClientID
	}
	switch target := value.(type) {
	case *polochon.Movie:
		if target == nil {
			return ErrInvalidArgument
		}
		return s.getMovieDetails(ctx, target)
	case *polochon.Show:
		if target == nil {
			return ErrInvalidArgument
		}
		show, err := s.getShowDetails(ctx, target)
		if err != nil {
			return err
		}
		*target = *show
		return nil
	case *polochon.ShowEpisode:
		if target == nil || target.Season <= 0 || target.Episode <= 0 {
			return ErrInvalidArgument
		}
		return s.getEpisodeDetails(ctx, target)
	default:
		return ErrInvalidArgument
	}
}

func (s *Simkl) lookup(ctx context.Context, kind string, params url.Values, title string, year int) (*mediaDetail, error) {
	if len(params) == 0 {
		if strings.TrimSpace(title) == "" {
			return nil, ErrInvalidArgument
		}
		params.Set("title", title)
		if year > 0 {
			params.Set("year", strconv.Itoa(year))
		}
	}
	id, err := s.client.resolve(ctx, kind, params)
	if err != nil {
		return nil, err
	}
	details, err := s.client.detail(ctx, kind, id)
	if err != nil {
		return nil, err
	}
	if imdb := params.Get("imdb"); imdb != "" {
		if details.IDs.Imdb != "" && details.IDs.Imdb != imdb {
			return nil, ErrInvalidResponse
		}
		if details.IDs.Imdb == "" {
			details.IDs.Imdb = imdb
		}
	}
	if details.IDs.Imdb == "" {
		return nil, ErrNotFound
	}
	return details, nil
}

func (s *Simkl) getMovieDetails(ctx context.Context, target *polochon.Movie) error {
	params := url.Values{}
	switch {
	case target.ImdbID != "":
		params.Set("imdb", target.ImdbID)
	case target.TmdbID > 0:
		params.Set("tmdb", strconv.Itoa(target.TmdbID))
	}
	details, err := s.lookup(ctx, "movies", params, target.Title, target.Year)
	if err != nil {
		return err
	}
	date, err := parseDate(details.Released)
	if err != nil {
		return err
	}
	movie := *target
	details.updateMovie(&movie)
	if details.Year == 0 && date != nil {
		movie.Year = date.Year()
	}
	if details.Runtime > 0 {
		movie.Runtime = details.Runtime
	}
	if details.Genres != nil {
		movie.Genres = details.Genres
	}
	if movie.ThumbURL == "" || movie.FanartURL == "" {
		return ErrMissingArtwork
	}
	*target = movie
	return nil
}

func (s *Simkl) lookupShow(ctx context.Context, target *polochon.Show) (*mediaDetail, error) {
	params := url.Values{}
	switch {
	case target.ImdbID != "":
		params.Set("imdb", target.ImdbID)
	case target.TvdbID > 0:
		params.Set("tvdb", strconv.Itoa(target.TvdbID))
	}
	return s.lookup(ctx, "tv", params, target.Title, target.Year)
}

func (c *apiClient) episodes(ctx context.Context, id int) ([]episode, error) {
	var items []episode
	path := "/tv/episodes/" + strconv.Itoa(id)
	if err := c.get(ctx, c.baseURL, path, nil, false, &items); err != nil {
		return nil, err
	}
	result := make([]episode, 0, len(items))
	seen := map[[2]int]bool{}
	for _, item := range items {
		key := [2]int{item.Season, item.Episode}
		if item.Type != "episode" || item.Season <= 0 || item.Episode <= 0 || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, item)
	}
	return result, nil
}

func (s *Simkl) getShowDetails(ctx context.Context, target *polochon.Show) (*polochon.Show, error) {
	details, err := s.lookupShow(ctx, target)
	if err != nil {
		return nil, err
	}
	date, err := parseDate(details.FirstAired)
	if err != nil {
		return nil, err
	}
	show := *target
	details.updateShow(&show)
	if date != nil {
		show.FirstAired = date
		if details.Year == 0 {
			show.Year = date.Year()
		}
	}
	if show.PosterURL == "" || show.FanartURL == "" || show.BannerURL == "" {
		return nil, ErrMissingArtwork
	}
	items, err := s.client.episodes(ctx, details.IDs.id())
	if err != nil {
		return nil, err
	}
	knownEpisodes := map[[2]int]*polochon.ShowEpisode{}
	for _, e := range target.Episodes {
		if e != nil {
			knownEpisodes[[2]int{e.Season, e.Episode}] = e
		}
	}
	show.Episodes = make([]*polochon.ShowEpisode, 0, len(items))
	for _, item := range items {
		date, err := parseDate(item.Date)
		if err != nil {
			return nil, err
		}
		e := polochon.NewShowEpisode(show.ShowConfig)
		e.Title, e.Plot = item.Title, item.Description
		e.ShowTitle, e.ShowImdbID, e.ShowTvdbID = show.Title, show.ImdbID, show.TvdbID
		e.Season, e.Episode = item.Season, item.Episode
		if known := knownEpisodes[[2]int{item.Season, item.Episode}]; known != nil {
			e.TvdbID, e.EpisodeImdbID = known.TvdbID, known.EpisodeImdbID
		}
		e.Runtime = item.Runtime
		if e.Runtime == 0 {
			e.Runtime = details.Runtime
		}
		if date != nil {
			e.Aired = date.Format(time.DateOnly)
		}
		e.ThumbURL = imageURL("episodes", item.Image, "_w")
		show.Episodes = append(show.Episodes, e)
	}
	return &show, nil
}

func (s *Simkl) getEpisodeDetails(ctx context.Context, target *polochon.ShowEpisode) error {
	show := polochon.Show{ShowConfig: target.ShowConfig}
	if parent := target.Show; parent != nil {
		// Explicit episode parent IDs take precedence over a stale cached show.
		if (target.ShowImdbID == "" || parent.ImdbID == "" || target.ShowImdbID == parent.ImdbID) &&
			(target.ShowTvdbID == 0 || parent.TvdbID == 0 || target.ShowTvdbID == parent.TvdbID) {
			show = *parent
		}
	}
	if show.Title == "" {
		show.Title = target.ShowTitle
	}
	if target.ShowImdbID != "" {
		show.ImdbID = target.ShowImdbID
	}
	if target.ShowTvdbID > 0 {
		show.TvdbID = target.ShowTvdbID
	}
	updated, err := s.getShowDetails(ctx, &show)
	if err != nil {
		return err
	}
	for _, item := range updated.Episodes {
		if item.Season != target.Season || item.Episode != target.Episode {
			continue
		}
		result := *target
		if item.Title != "" {
			result.Title = item.Title
		}
		if item.Plot != "" {
			result.Plot = item.Plot
		}
		if item.Aired != "" {
			result.Aired = item.Aired
		}
		result.ShowTitle, result.ShowImdbID, result.ShowTvdbID = item.ShowTitle, item.ShowImdbID, item.ShowTvdbID
		if item.Runtime > 0 {
			result.Runtime = item.Runtime
		}
		if item.ThumbURL != "" {
			result.ThumbURL = item.ThumbURL
		}
		result.Show = updated
		*target = result
		return nil
	}
	return ErrNotFound
}
