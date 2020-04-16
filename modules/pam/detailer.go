package pam

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/papi"
	"github.com/sirupsen/logrus"
)

// GetDetails implements the Detailer interface
func (p *Pam) GetDetails(i interface{}, log *logrus.Entry) error {
	switch resource := i.(type) {
	case *polochon.Movie:
		m := &papi.Movie{Movie: resource}
		return p.client.GetDetails(m)
	case *polochon.Show:
		s := &papi.Show{Show: resource}
		return p.client.GetDetails(s)
	case *polochon.ShowEpisode:
		e := &papi.Episode{ShowEpisode: resource}
		return p.client.GetDetails(e)
	default:
		return ErrInvalidType
	}
}
