package tvdb

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agnivade/levenshtein"
	"github.com/goccy/go-yaml"

	polochon "github.com/odwrtw/polochon/lib"
)

var (
	_ polochon.Detailer = (*TvDB)(nil)
	_ polochon.Calendar = (*TvDB)(nil)
)

func init() {
	polochon.RegisterModule(&TvDB{})
}

const (
	moduleName = "tvdb"

	seriesBannerArtwork     = 1
	seriesPosterArtwork     = 2
	seriesBackgroundArtwork = 3
)

var (
	ErrShowNotFound                   = errors.New("tvdb: show not found")
	ErrShowImageNotFound              = errors.New("tvdb: show image not found")
	ErrNotEnoughArguments             = errors.New("tvdb: not enough arguments to perform search")
	ErrInvalidArgument                = errors.New("tvdb: invalid argument type")
	ErrMissingAPIKey                  = errors.New("tvdb: missing API key")
	ErrMissingShowEpisodeInformations = errors.New("tvdb: missing show episode informations to get details")
	ErrFailedToUpdateEpisode          = errors.New("tvdb: failed to update episode details")
)

// Params represents the module parameters for TheTVDB API v4.
type Params struct {
	APIKey string `yaml:"api_key"`
	Pin    string `yaml:"pin"`
}

// TvDB implements the Detailer and Calendar interfaces using TheTVDB API v4.
type TvDB struct {
	client     *apiClient
	configured bool
}

func (t *TvDB) Init(p []byte, _ *slog.Logger) error {
	if t.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}
	return t.InitWithParams(params)
}

func (t *TvDB) InitWithParams(params *Params) error {
	if params.APIKey == "" {
		return ErrMissingAPIKey
	}

	t.client = newAPIClient(params.APIKey, params.Pin)
	t.configured = true
	return nil
}

func (t *TvDB) Name() string {
	return moduleName
}

func (t *TvDB) Status() (polochon.ModuleStatus, error) {
	_, err := t.searchByImdbID(context.Background(), "tt2085059")
	if err != nil {
		return polochon.StatusFail, err
	}
	return polochon.StatusOK, nil
}

func (t *TvDB) searchByImdbID(ctx context.Context, id string) (*series, error) {
	shows, err := t.client.searchSeriesByRemoteID(ctx, id)
	if err != nil {
		if isHTTPStatus(err, http.StatusNotFound) {
			return nil, ErrShowNotFound
		}
		return nil, err
	}
	if len(shows) == 0 {
		return nil, ErrShowNotFound
	}
	return &shows[0], nil
}

func (t *TvDB) searchByTvdbID(ctx context.Context, id int) (*series, error) {
	show, err := t.client.getSeries(ctx, id)
	if err != nil {
		if isHTTPStatus(err, http.StatusNotFound) {
			return nil, ErrShowNotFound
		}
		return nil, err
	}
	return show, nil
}

func (t *TvDB) searchByName(ctx context.Context, query string, year int) (*series, error) {
	shows, err := t.client.searchSeries(ctx, query, year)
	if err != nil {
		if isHTTPStatus(err, http.StatusNotFound) {
			return nil, ErrShowNotFound
		}
		return nil, err
	}

	show := bestShowMatch(shows, query, year)
	if show == nil {
		return nil, ErrShowNotFound
	}
	return show, nil
}

func bestShowMatch(shows []series, query string, year int) *series {
	if year != 0 {
		for _, show := range shows {
			if strings.EqualFold(show.Name, query) && showYear(show) == year {
				return &show
			}
		}

		for _, show := range shows {
			if showYear(show) != year {
				continue
			}
			for _, alias := range show.Aliases {
				if strings.EqualFold(alias.Name, query) {
					return &show
				}
			}
		}
	}

	for _, show := range shows {
		if strings.EqualFold(show.Name, query) {
			return &show
		}
	}
	for _, show := range shows {
		for _, alias := range show.Aliases {
			if strings.EqualFold(alias.Name, query) {
				return &show
			}
		}
	}

	var bestMatch *series
	var bestDistance int
	for _, show := range shows {
		distance := levenshtein.ComputeDistance(
			strings.ToLower(show.Name),
			strings.ToLower(query),
		)
		if bestMatch == nil || distance < bestDistance ||
			(year != 0 && distance == bestDistance && showYear(show) == year && showYear(*bestMatch) != year) {
			bestMatch = &show
			bestDistance = distance
		}
	}
	return bestMatch
}

func showYear(show series) int {
	year, _ := strconv.Atoi(show.Year)
	if year != 0 {
		return year
	}
	date, err := time.Parse("2006-01-02", show.FirstAired)
	if err != nil {
		return 0
	}
	return date.Year()
}

func (t *TvDB) GetDetails(ctx context.Context, value any) error {
	switch video := value.(type) {
	case *polochon.Show:
		return t.getShowDetails(ctx, video, 0, 0)
	case *polochon.ShowEpisode:
		return t.getEpisodeDetails(ctx, video)
	default:
		return ErrInvalidArgument
	}
}

func (t *TvDB) searchShow(ctx context.Context, show *polochon.Show) (*series, error) {
	switch {
	case show.TvdbID != 0:
		return t.searchByTvdbID(ctx, show.TvdbID)
	case show.ImdbID != "":
		return t.searchByImdbID(ctx, show.ImdbID)
	case show.Title != "":
		return t.searchByName(ctx, show.Title, show.Year)
	default:
		return nil, ErrNotEnoughArguments
	}
}

func (t *TvDB) getShowEpisodes(
	ctx context.Context,
	show *polochon.Show,
	details *series,
	season, number int,
) error {
	episodes, err := t.client.getEpisodes(ctx, details.ID, season, number)
	if err != nil {
		return err
	}

	show.Episodes = make([]*polochon.ShowEpisode, 0, len(episodes))
	for _, item := range episodes {
		runtime := 0
		if item.Runtime != nil {
			runtime = *item.Runtime
		} else if details.AverageRuntime != nil {
			runtime = *details.AverageRuntime
		}

		episode := polochon.NewShowEpisode(polochon.ShowConfig{})
		episode.Title = item.Name
		episode.ShowTitle = show.Title
		episode.Season = item.SeasonNumber
		episode.Episode = item.Number
		episode.TvdbID = item.ID
		episode.Aired = item.Aired
		episode.Plot = item.Overview
		episode.Runtime = runtime
		episode.ThumbURL = item.Image
		episode.ShowImdbID = show.ImdbID
		episode.ShowTvdbID = show.TvdbID
		episode.EpisodeImdbID = imdbID(item.RemoteIDs)
		show.Episodes = append(show.Episodes, episode)
	}
	return nil
}

func (t *TvDB) getShowImages(show *polochon.Show, details *series) error {
	show.BannerURL = bestArtwork(details.Artworks, seriesBannerArtwork)
	show.PosterURL = bestArtwork(details.Artworks, seriesPosterArtwork)
	show.FanartURL = bestArtwork(details.Artworks, seriesBackgroundArtwork)

	if show.PosterURL == "" {
		show.PosterURL = details.Image
	}
	if show.BannerURL == "" || show.PosterURL == "" || show.FanartURL == "" {
		return ErrShowImageNotFound
	}
	return nil
}

func bestArtwork(artworks []artwork, artworkType int) string {
	var best *artwork
	for i := range artworks {
		candidate := &artworks[i]
		if candidate.Type != artworkType || candidate.Image == "" {
			continue
		}
		if best == nil || candidate.Score > best.Score {
			best = candidate
		}
	}
	if best == nil {
		return ""
	}
	return best.Image
}

func (t *TvDB) getShowDetails(
	ctx context.Context,
	show *polochon.Show,
	season, number int,
) error {
	found, err := t.searchShow(ctx, show)
	if err != nil {
		return err
	}

	details, err := t.client.getSeries(ctx, found.ID)
	if err != nil {
		return err
	}

	show.Title = details.Name
	show.Plot = details.Overview
	show.TvdbID = details.ID
	if id := imdbID(details.RemoteIDs); id != "" {
		show.ImdbID = id
	}
	if details.FirstAired != "" {
		date, err := time.Parse("2006-01-02", details.FirstAired)
		if err != nil {
			return err
		}
		show.Year = date.Year()
		show.FirstAired = &date
	}

	if err := t.getShowImages(show, details); err != nil {
		return err
	}
	return t.getShowEpisodes(ctx, show, details, season, number)
}

func (t *TvDB) getEpisodeDetails(ctx context.Context, target *polochon.ShowEpisode) error {
	if target.Season == 0 || target.Episode == 0 {
		return ErrMissingShowEpisodeInformations
	}
	if target.ShowTitle == "" && target.ShowImdbID == "" {
		return ErrMissingShowEpisodeInformations
	}

	show := target.Show
	if show == nil {
		show = polochon.NewShow(polochon.ShowConfig{})
	}
	if show.Title == "" {
		show.Title = target.ShowTitle
	}
	if show.ImdbID == "" {
		show.ImdbID = target.ShowImdbID
	}
	if err := t.getShowDetails(ctx, show, target.Season, target.Episode); err != nil {
		return err
	}

	for _, item := range show.Episodes {
		if item.Season != target.Season || item.Episode != target.Episode {
			continue
		}
		target.Title = item.Title
		target.ShowTitle = item.ShowTitle
		target.Season = item.Season
		target.Episode = item.Episode
		target.TvdbID = item.TvdbID
		target.Aired = item.Aired
		target.Plot = item.Plot
		target.Runtime = item.Runtime
		target.ThumbURL = item.ThumbURL
		target.Rating = item.Rating
		target.ShowImdbID = item.ShowImdbID
		target.ShowTvdbID = item.ShowTvdbID
		target.EpisodeImdbID = item.EpisodeImdbID
		target.Show = show
		return nil
	}
	return ErrFailedToUpdateEpisode
}

func imdbID(ids []remoteID) string {
	for _, id := range ids {
		if strings.EqualFold(id.SourceName, "IMDB") || strings.HasPrefix(id.ID, "tt") {
			return id.ID
		}
	}
	return ""
}

func isHTTPStatus(err error, status int) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.statusCode == status
}

func (t *TvDB) GetShowCalendar(ctx context.Context, show *polochon.Show) (*polochon.ShowCalendar, error) {
	if err := t.getShowDetails(ctx, show, 0, 0); err != nil {
		return nil, err
	}
	if show == nil || show.ImdbID == "" {
		return nil, polochon.ErrCalendarNotFound
	}

	calendar := polochon.NewShowCalendar(show.ImdbID)
	for _, episode := range show.Episodes {
		var airedDate *time.Time
		if episode.Aired != "" {
			aired, err := time.Parse("2006-01-02", episode.Aired)
			if err != nil {
				return nil, err
			}
			airedDate = &aired
		}
		calendar.Episodes = append(calendar.Episodes, &polochon.ShowCalendarEpisode{
			Season:    episode.Season,
			Episode:   episode.Episode,
			AiredDate: airedDate,
		})
	}
	return calendar, nil
}
