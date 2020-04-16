package papi

import "sort"

// ShowCollection represent a collection of shows
type ShowCollection struct {
	shows map[string]*Show
}

// NewShowCollection returns a new empty collection of shows
func NewShowCollection() *ShowCollection {
	return &ShowCollection{
		shows: map[string]*Show{},
	}
}

// List return a list of shows
func (s *ShowCollection) List() []*Show {
	shows := make([]*Show, len(s.shows))

	i := 0
	for _, v := range s.shows {
		shows[i] = v
		i++
	}

	// Sort the slice by Imdb ID
	sort.Slice(shows, func(i, j int) bool {
		return shows[i].ImdbID < shows[j].ImdbID
	})

	return shows
}

// Has checks if a show is in a collection
func (s *ShowCollection) Has(imdbID string) (*Show, bool) {
	show, ok := s.shows[imdbID]
	return show, ok
}

// Add adds a show to the collection
func (s *ShowCollection) Add(show *Show) error {
	if show == nil || show.Show == nil {
		return ErrMissingShow
	}

	if show.ImdbID == "" {
		return ErrMissingShowID
	}

	s.shows[show.ImdbID] = show

	return nil
}
