package pam

import (
	"github.com/odwrtw/papi"
	"github.com/odwrtw/polochon/lib"
)

func moviePolochonToPapi(movie *polochon.Movie) (*papi.Movie, error) {
	if movie.ImdbID == "" {
		return nil, ErrMissingImdbID
	}

	return &papi.Movie{ImdbID: movie.ImdbID}, nil
}

func moviePapiToPolochon(papiMovie *papi.Movie, polochonMovie *polochon.Movie) {
	polochonMovie.OriginalTitle = papiMovie.OriginalTitle
	polochonMovie.Plot = papiMovie.Plot
	polochonMovie.Rating = papiMovie.Rating
	polochonMovie.Runtime = papiMovie.Runtime
	polochonMovie.SortTitle = papiMovie.SortTitle
	polochonMovie.Tagline = papiMovie.Tagline
	polochonMovie.Thumb = papiMovie.Thumb
	polochonMovie.Fanart = papiMovie.Fanart
	polochonMovie.Title = papiMovie.Title
	polochonMovie.TmdbID = papiMovie.TmdbID
	polochonMovie.Votes = papiMovie.Votes
	polochonMovie.Year = papiMovie.Year
	polochonMovie.Genres = papiMovie.Genres

}

func (p *Pam) getMovieDetails(movie *polochon.Movie) error {
	papiMovie, err := moviePolochonToPapi(movie)
	if err != nil {
		return err
	}

	// Get the details of the resource
	if err := p.client.GetDetails(papiMovie); err != nil {
		return err
	}

	moviePapiToPolochon(papiMovie, movie)

	return nil
}
