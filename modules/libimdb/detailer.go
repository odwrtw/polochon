package libimdb

import (
	"context"

	imdb "github.com/odwrtw/libimdb"

	polochon "github.com/odwrtw/polochon/lib"
)

// GetDetails supplies safe identity and canonical-name hints. It never picks
// an identity from fuzzy candidates or replaces supplied IMDb/provider IDs.
// Call polochon.GetDetails to continue with provider enrichment.
func (c *Catalog) GetDetails(ctx context.Context, value any) error {
	query, id, err := detailQuery(value)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	catalog, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = catalog.Close() }()

	var title imdb.Title
	if id != "" {
		title, err = catalog.Lookup(ctx, id)
	} else {
		var match imdb.Resolution
		match, err = catalog.Resolve(ctx, query)
		title = match.Title
	}
	if err != nil {
		return err
	}
	if title.Kind != query.Kind {
		return ErrKindMismatch
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	applyHints(value, title)
	c.log.Debug("catalog supplied matching hints", "imdb_id", title.ID, "kind", title.Kind, "title", title.PrimaryTitle)
	return nil
}

func detailQuery(value any) (imdb.Query, string, error) {
	switch v := value.(type) {
	case *polochon.Movie:
		if v == nil {
			break
		}
		if v.ImdbID == "" && v.TmdbID != 0 {
			return imdb.Query{}, "", polochon.ErrNotAvailable
		}
		return imdb.Query{Title: v.Title, StartYear: v.Year, Kind: imdb.Movie}, v.ImdbID, nil
	case *polochon.Show:
		if v == nil {
			break
		}
		if v.ImdbID == "" && v.TvdbID != 0 {
			return imdb.Query{}, "", polochon.ErrNotAvailable
		}
		return imdb.Query{Title: v.Title, StartYear: v.Year, Kind: imdb.Show}, v.ImdbID, nil
	case *polochon.ShowEpisode:
		if v == nil {
			break
		}
		q := imdb.Query{Title: v.ShowTitle, Kind: imdb.Show}
		id := v.ShowImdbID
		hasProviderID := v.ShowTvdbID != 0
		if v.Show != nil {
			if id != "" && v.Show.ImdbID != "" && id != v.Show.ImdbID {
				return imdb.Query{}, "", ErrConflictingIDs
			}
			if id == "" {
				id = v.Show.ImdbID
			}
			if q.Title == "" {
				q.Title = v.Show.Title
			}
			q.StartYear = v.Show.Year
			hasProviderID = hasProviderID || v.Show.TvdbID != 0
		}
		if id == "" && hasProviderID {
			return imdb.Query{}, "", polochon.ErrNotAvailable
		}
		return q, id, nil
	}
	return imdb.Query{}, "", ErrInvalidArgument
}

func applyHints(value any, title imdb.Title) {
	switch v := value.(type) {
	case *polochon.Movie:
		v.ImdbID = title.ID
		v.Title = title.PrimaryTitle
		if v.OriginalTitle == "" {
			v.OriginalTitle = title.OriginalTitle
		}
		if v.Year == 0 && title.StartYear != nil {
			v.Year = *title.StartYear
		}
	case *polochon.Show:
		showHints(v, title)
	case *polochon.ShowEpisode:
		v.ShowImdbID = title.ID
		v.ShowTitle = title.PrimaryTitle
		// Do not manufacture a partially detailed Show: library.addShow
		// treats a supplied Show as already enriched.
		if v.Show != nil {
			showHints(v.Show, title)
		}
	}
}

func showHints(show *polochon.Show, title imdb.Title) {
	show.ImdbID = title.ID
	show.Title = title.PrimaryTitle
	if show.Year == 0 && title.StartYear != nil {
		show.Year = *title.StartYear
	}
}
