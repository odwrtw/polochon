package tmdb

import (
	"errors"
	"strconv"
	"time"

	"github.com/agnivade/levenshtein"
	tmdb "github.com/cyruzin/golang-tmdb"
	polochon "github.com/odwrtw/polochon/lib"
)

// These wrappers keep TV calls replaceable by unit tests, as the movie calls
// are in tmdb.go.
var (
	tmdbSearchTV = func(t *tmdb.Client, title string, options map[string]string) (*tmdb.SearchTVShows, error) {
		return t.GetSearchTVShow(title, options)
	}
	tmdbSearchTVShow = func(t *tmdb.Client, title string, options map[string]string) (*tmdb.SearchTVShows, error) {
		return tmdbSearchTV(t, title, options)
	}
	tmdbSearchByExternalID = func(t *tmdb.Client, id, source string, options map[string]string) (*tmdb.FindByID, error) {
		if options == nil {
			options = map[string]string{}
		}
		options["external_source"] = source
		return t.GetFindByID(id, options)
	}
	tmdbFindTVByExternalID = func(t *tmdb.Client, id, source string, options map[string]string) (*tmdb.FindByID, error) {
		return tmdbSearchByExternalID(t, id, source, options)
	}
	tmdbGetTVDetails = func(t *tmdb.Client, id int, options map[string]string) (*tmdb.TVDetails, error) {
		return t.GetTVDetails(id, options)
	}
	tmdbGetTVInfo = func(t *tmdb.Client, id int, options map[string]string) (*tmdb.TVDetails, error) {
		return tmdbGetTVDetails(t, id, options)
	}
	tmdbGetTVExternalIDs = func(t *tmdb.Client, id int, options map[string]string) (*tmdb.TVExternalIDs, error) {
		return t.GetTVExternalIDs(id, options)
	}
	tmdbGetEpisodeInfo = func(t *tmdb.Client, id, season, episode int, options map[string]string) (*tmdb.TVEpisodeDetails, error) {
		return t.GetTVEpisodeDetails(id, season, episode, options)
	}
	tmdbGetTVEpisodeInfo = func(t *tmdb.Client, id, season, episode int, options map[string]string) (*tmdb.TVEpisodeDetails, error) {
		return tmdbGetEpisodeInfo(t, id, season, episode, options)
	}
	tmdbGetEpisodeExternalIDs = func(t *tmdb.Client, id, season, episode int) (*tmdb.TVEpisodeExternalIDs, error) {
		return t.GetTVEpisodeExternalIDs(id, season, episode)
	}
	tmdbGetTVEpisodeExternalIDs = func(t *tmdb.Client, id, season, episode int) (*tmdb.TVEpisodeExternalIDs, error) {
		return tmdbGetEpisodeExternalIDs(t, id, season, episode)
	}
)

var (
	// ErrIdentityConflict means that supplied identifiers refer to different
	// TMDB objects. The input metadata is left untouched in this case.
	ErrIdentityConflict               = errors.New("tmdb: conflicting identities")
	ErrConflictingIDs                 = ErrIdentityConflict
	ErrNoShowFound                    = errors.New("tmdb: show not found")
	ErrNoShowTitle                    = errors.New("tmdb: can not search for a show with no title")
	ErrNoShowImDBID                   = errors.New("tmdb: can not search for a show with no imdb")
	ErrNoShowTvDBID                   = errors.New("tmdb: can not search for a show with no tvdb")
	ErrMissingShowEpisodeInformations = errors.New("tmdb: missing show episode informations")
	ErrFailedToGetShowDetails         = errors.New("tmdb: failed to get show details")
	ErrFailedToGetEpisodeDetails      = errors.New("tmdb: failed to get episode details")
)

// Common spelling aliases for callers using the names from the other modules.
var (
	ErrIdentityConflictTmdb = ErrIdentityConflict
	ErrNoShowImdbID         = ErrNoShowImDBID
	ErrNoShowTvdbID         = ErrNoShowTvDBID
)

type resolvedShow struct {
	id       int
	details  *tmdb.TVDetails
	external *tmdb.TVExternalIDs
}

func imageURL(path string) string {
	if path == "" {
		return ""
	}
	return TmDBimageBaseURL + path
}

func parseTMDBDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

func (t *TmDB) findTVByExternalID(id, source string) (int, error) {
	if id == "" {
		switch source {
		case "imdb_id":
			return 0, ErrNoShowImDBID
		case "tvdb_id":
			return 0, ErrNoShowTvDBID
		}
		return 0, ErrNoShowFound
	}

	result, err := tmdbFindTVByExternalID(t.client, id, source, map[string]string{})
	if err != nil {
		return 0, err
	}
	if result == nil || len(result.TvResults) == 0 {
		return 0, ErrNoShowFound
	}
	return int(result.TvResults[0].ID), nil
}

func (t *TmDB) findTVByTitle(show *polochon.Show) (int, error) {
	if show.Title == "" {
		return 0, ErrNoShowTitle
	}

	options := map[string]string{}
	if show.Year != 0 {
		options["first_air_date_year"] = strconv.Itoa(show.Year)
	}
	result, err := tmdbSearchTVShow(t.client, show.Title, options)
	if err != nil {
		return 0, err
	}
	if result == nil || result.SearchTVShowsResults == nil || len(result.Results) == 0 {
		return 0, ErrNoShowFound
	}

	var best tmdb.TVShowResult
	minDistance := -1
	for _, candidate := range result.Results {
		distance := levenshtein.ComputeDistance(show.Title, candidate.Name)
		if minDistance == -1 || distance < minDistance {
			minDistance = distance
			best = candidate
		}
	}
	return int(best.ID), nil
}

func (t *TmDB) resolveTV(show *polochon.Show) (*resolvedShow, error) {
	id := show.TmdbID
	if id == 0 {
		var err error
		if show.ImdbID != "" {
			id, err = t.findTVByExternalID(show.ImdbID, "imdb_id")
			if err != nil && !errors.Is(err, ErrNoShowFound) {
				return nil, err
			}
		}
		if id == 0 && show.TvdbID != 0 {
			id, err = t.findTVByExternalID(strconv.Itoa(show.TvdbID), "tvdb_id")
			if err != nil && !errors.Is(err, ErrNoShowFound) {
				return nil, err
			}
		}
		if id == 0 {
			id, err = t.findTVByTitle(show)
			if err != nil {
				return nil, err
			}
		}
	}

	details, err := tmdbGetTVInfo(t.client, id, map[string]string{})
	if err != nil {
		return nil, err
	}
	if details == nil {
		return nil, ErrFailedToGetShowDetails
	}
	if details.ID != 0 && int(details.ID) != id {
		return nil, ErrIdentityConflict
	}

	external, err := tmdbGetTVExternalIDs(t.client, id, map[string]string{})
	if err != nil {
		return nil, err
	}
	if external == nil {
		return nil, ErrFailedToGetShowDetails
	}

	if show.ImdbID != "" && external.IMDbID != "" && show.ImdbID != external.IMDbID {
		return nil, ErrIdentityConflict
	}
	if show.TvdbID != 0 && external.TVDBID != 0 && show.TvdbID != int(external.TVDBID) {
		return nil, ErrIdentityConflict
	}

	return &resolvedShow{id: id, details: details, external: external}, nil
}

func (t *TmDB) getShowDetails(show *polochon.Show) error {
	resolved, err := t.resolveTV(show)
	if err != nil {
		return err
	}
	firstAired, err := parseTMDBDate(resolved.details.FirstAirDate)
	if err != nil {
		return err
	}

	updated := *show
	updated.TmdbID = resolved.id
	updated.ImdbID = resolved.external.IMDbID
	if updated.ImdbID == "" {
		updated.ImdbID = show.ImdbID
	}
	updated.TvdbID = int(resolved.external.TVDBID)
	if updated.TvdbID == 0 {
		updated.TvdbID = show.TvdbID
	}
	updated.Title = resolved.details.Name
	updated.Plot = resolved.details.Overview
	updated.Year = 0
	if firstAired != nil {
		updated.Year = firstAired.Year()
	}
	updated.FirstAired = firstAired
	updated.Rating = resolved.details.VoteAverage
	updated.URL = resolved.details.Homepage
	updated.PosterURL = imageURL(resolved.details.PosterPath)
	updated.FanartURL = imageURL(resolved.details.BackdropPath)
	updated.BannerURL = updated.FanartURL
	// TMDB show details do not include individual episode details here. In
	// particular, never fan out over the seasons returned by the API.
	updated.Episodes = show.Episodes
	*show = updated
	return nil
}

func (t *TmDB) getEpisodeDetails(episode *polochon.ShowEpisode) error {
	missingParent := episode.Show == nil && episode.ShowTitle == "" && episode.ShowImdbID == "" && episode.ShowTvdbID == 0
	if episode.Season <= 0 || episode.Episode <= 0 {
		if missingParent {
			return ErrInvalidArgument
		}
		return ErrMissingShowEpisodeInformations
	}
	if missingParent {
		return ErrMissingShowEpisodeInformations
	}

	var parent polochon.Show
	if episode.Show != nil {
		parent = *episode.Show
	}
	if parent.Title == "" {
		parent.Title = episode.ShowTitle
	}
	if parent.ImdbID == "" {
		parent.ImdbID = episode.ShowImdbID
	}
	if parent.TvdbID == 0 {
		parent.TvdbID = episode.ShowTvdbID
	}
	parent.ShowConfig = episode.ShowConfig

	if err := t.getShowDetails(&parent); err != nil {
		return err
	}

	details, err := tmdbGetTVEpisodeInfo(t.client, parent.TmdbID, episode.Season, episode.Episode, map[string]string{})
	if err != nil {
		return err
	}
	if details == nil {
		return ErrFailedToGetEpisodeDetails
	}
	external, err := tmdbGetTVEpisodeExternalIDs(t.client, parent.TmdbID, episode.Season, episode.Episode)
	if err != nil {
		return err
	}
	if external == nil {
		return ErrFailedToGetEpisodeDetails
	}
	if episode.EpisodeImdbID != "" && external.IMDbID != "" && episode.EpisodeImdbID != external.IMDbID {
		return ErrIdentityConflict
	}
	if episode.TvdbID != 0 && external.TVDBID != 0 && episode.TvdbID != int(external.TVDBID) {
		return ErrIdentityConflict
	}

	updated := *episode
	updated.Title = details.Name
	updated.ShowTitle = parent.Title
	if details.SeasonNumber != 0 {
		updated.Season = details.SeasonNumber
	}
	if details.EpisodeNumber != 0 {
		updated.Episode = details.EpisodeNumber
	}
	updated.TvdbID = int(external.TVDBID)
	if updated.TvdbID == 0 {
		updated.TvdbID = episode.TvdbID
	}
	updated.Aired = details.AirDate
	updated.Plot = details.Overview
	updated.Runtime = details.Runtime
	updated.ThumbURL = imageURL(details.StillPath)
	updated.Rating = details.VoteAverage
	updated.ShowImdbID = parent.ImdbID
	updated.ShowTvdbID = parent.TvdbID
	updated.EpisodeImdbID = external.IMDbID
	if updated.EpisodeImdbID == "" {
		updated.EpisodeImdbID = episode.EpisodeImdbID
	}
	// ShowTmdbID deliberately is not part of ShowEpisode. Keep the resolved
	// show available only through the non-persisted Show pointer when supplied.
	if episode.Show != nil {
		updated.Show = &parent
	}
	*episode = updated
	return nil
}
