package simkl

import (
	"context"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

func (s *Simkl) details(ctx context.Context, kind, id string) (record, error) {
	var r record
	err := s.client.get(ctx, s.client.api+"/"+kind+"/"+id, false, time.Hour, false, &r)
	if err != nil {
		return r, err
	}
	if r.IDs.simkl() == "" {
		return r, ErrNotFound
	}
	if r.IDs.simkl() != id || !r.compatible(kind) {
		return r, ErrUnsupported
	}
	s.client.remember(kind, r)
	if _, err = s.client.known(kind, "imdb", r.IDs.IMDb); err != nil {
		return r, err
	}
	return r, nil
}

func (s *Simkl) episodes(ctx context.Context, id string) ([]episodeRecord, error) {
	var all []episodeRecord
	if err := s.client.get(ctx, s.client.api+"/tv/episodes/"+id, false, time.Hour, false, &all); err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	for _, e := range all {
		if !e.validCoordinates() {
			continue
		}
		key := episodeKey(e)
		if seen[key] {
			return nil, ErrAmbiguous
		}
		seen[key] = true
	}
	valid := make([]episodeRecord, 0, len(all))
	for _, e := range all {
		if e.validCoordinates() {
			valid = append(valid, e)
		}
	}
	return valid, nil
}

func (s *Simkl) showDetails(ctx context.Context, show *polochon.Show) (record, []episodeRecord, error) {
	if show == nil {
		return record{}, nil, ErrUnsupported
	}
	id, err := s.resolve(ctx, "tv", show.ImdbID, show.TvdbID, show.Title, show.Year)
	if err != nil {
		return record{}, nil, err
	}
	r, err := s.details(ctx, "tv", id)
	if err != nil {
		return r, nil, err
	}
	if show.ImdbID != "" && show.ImdbID != r.IDs.IMDb {
		return r, nil, ErrAmbiguous
	}
	episodes, err := s.episodes(ctx, id)
	if err != nil {
		return r, nil, err
	}
	r.show(show)
	show.Episodes = make([]*polochon.ShowEpisode, 0, len(episodes))
	for _, e := range episodes {
		show.Episodes = append(show.Episodes, e.episode(show))
	}
	return r, episodes, nil
}

func (s *Simkl) GetDetails(ctx context.Context, target any) error {
	if _, err := s.ready(); err != nil {
		return err
	}
	switch v := target.(type) {
	case *polochon.Movie:
		if v == nil {
			return ErrUnsupported
		}
		id, err := s.resolve(ctx, "movies", v.ImdbID, v.TmdbID, v.Title, v.Year)
		if err != nil {
			return err
		}
		r, err := s.details(ctx, "movies", id)
		if err != nil {
			return err
		}
		if v.ImdbID != "" && v.ImdbID != r.IDs.IMDb {
			return ErrAmbiguous
		}
		r.movie(v)
		return nil
	case *polochon.Show:
		_, _, err := s.showDetails(ctx, v)
		return err
	case *polochon.ShowEpisode:
		if v == nil || v.Season < 0 || v.Episode <= 0 {
			return ErrUnsupported
		}
		show := polochon.NewShow(v.ShowConfig)
		if v.Show != nil {
			*show = *v.Show
		}
		if v.ShowImdbID != "" {
			if show.ImdbID != "" && show.ImdbID != v.ShowImdbID {
				return ErrAmbiguous
			}
			show.ImdbID = v.ShowImdbID
		}
		if show.TvdbID == 0 {
			show.TvdbID = v.ShowTvdbID
		}
		if show.Title == "" {
			show.Title = v.ShowTitle
		}
		_, _, err := s.showDetails(ctx, show)
		if err != nil {
			return err
		}
		for _, ep := range show.Episodes {
			if ep.Season == v.Season && ep.Episode == v.Episode {
				if ep.Title != "" {
					v.Title = ep.Title
				}
				if ep.Plot != "" {
					v.Plot = ep.Plot
				}
				if ep.Aired != "" {
					v.Aired = ep.Aired
				}
				if ep.Runtime > 0 {
					v.Runtime = ep.Runtime
				}
				if ep.ThumbURL != "" {
					v.ThumbURL = ep.ThumbURL
				}
				if ep.EpisodeImdbID != "" {
					v.EpisodeImdbID = ep.EpisodeImdbID
				}
				if ep.TvdbID > 0 {
					v.TvdbID = ep.TvdbID
				}
				v.Show = show
				v.ShowImdbID = show.ImdbID
				v.ShowTvdbID = show.TvdbID
				v.ShowTitle = show.Title
				return nil
			}
		}
		return ErrNotFound
	default:
		return ErrUnsupported
	}
}

func (s *Simkl) GetShowCalendar(ctx context.Context, show *polochon.Show) (*polochon.ShowCalendar, error) {
	if _, err := s.ready(); err != nil {
		return nil, err
	}
	r, episodes, err := s.showDetails(ctx, show)
	if err != nil {
		return nil, err
	}
	calendar := polochon.NewShowCalendar(show.ImdbID)
	for _, ep := range episodes {
		if ep.Aired != nil && !*ep.Aired {
			continue
		}
		date, ok := availableDate(ep.Date, r.Airs.Timezone, s.client.now())
		if !ok {
			continue
		}
		calendar.Episodes = append(calendar.Episodes, &polochon.ShowCalendarEpisode{Season: ep.Season.integer(), Episode: ep.Episode.integer(), AiredDate: &date})
	}
	return calendar, nil
}

func availableDate(value, timezone string, now time.Time) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, !t.After(now)
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", value, loc)
	if err != nil {
		return time.Time{}, false
	}
	// Use AddDate, not 24h: a release day may span a DST transition.
	eligible := t.AddDate(0, 0, 1)
	return eligible, !eligible.After(now)
}
