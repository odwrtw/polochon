package simkl

import (
	"context"
	"errors"

	polochon "github.com/odwrtw/polochon/lib"
)

var _ polochon.Calendar = (*Simkl)(nil)

// GetShowCalendar uses the full episode list, not the rolling global calendar.
// It does not depend on artwork or mutate the supplied show.
func (s *Simkl) GetShowCalendar(ctx context.Context, show *polochon.Show) (*polochon.ShowCalendar, error) {
	if show == nil {
		return nil, ErrInvalidArgument
	}
	if s.client == nil {
		return nil, ErrMissingClientID
	}
	details, err := s.lookupShow(ctx, show)
	if errors.Is(err, ErrNotFound) {
		return nil, polochon.ErrCalendarNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := s.client.episodes(ctx, details.IDs.id())
	if err != nil {
		return nil, err
	}
	calendar := polochon.NewShowCalendar(details.IDs.Imdb)
	for _, item := range items {
		date, err := parseDate(item.Date)
		if err != nil {
			return nil, err
		}
		// Polochon treats nil dates as already available. Only use that for
		// episodes Simkl explicitly marks aired, never for undated premieres.
		if date == nil && !item.Aired {
			continue
		}
		calendar.Episodes = append(calendar.Episodes, &polochon.ShowCalendarEpisode{
			Season: item.Season, Episode: item.Episode, AiredDate: date,
		})
	}
	return calendar, nil
}
