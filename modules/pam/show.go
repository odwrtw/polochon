package pam

import (
	"github.com/odwrtw/papi"
	"github.com/odwrtw/polochon/lib"
)

func showPolochonToPapi(show *polochon.Show) (*papi.Show, error) {
	if show.ImdbID == "" {
		return nil, ErrMissingImdbID
	}

	return &papi.Show{ImdbID: show.ImdbID}, nil
}

func showPapiToPolochon(papiShow *papi.Show, polochonShow *polochon.Show) {
	polochonShow.Title = papiShow.Title
	polochonShow.Rating = papiShow.Rating
	polochonShow.Plot = papiShow.Plot
	polochonShow.TvdbID = papiShow.TvdbID
	polochonShow.ImdbID = papiShow.ImdbID
	polochonShow.Year = papiShow.Year
	polochonShow.FirstAired = papiShow.FirstAired
}

func (p *Pam) getShowDetails(show *polochon.Show) error {
	papiShow, err := showPolochonToPapi(show)
	if err != nil {
		return err
	}

	// Get the details of the resource
	if err := p.client.GetDetails(papiShow); err != nil {
		return err
	}

	showPapiToPolochon(papiShow, show)

	return nil
}
