// Package pam - Polochon api module
package pam

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetDetails implements the Detailer interface
func (p *Pam) GetDetails(i interface{}, log *logrus.Entry) error {
	switch resource := i.(type) {
	case *polochon.Movie:
		return p.getMovieDetails(resource)
	case *polochon.Show:
		return p.getShowDetails(resource)
	case *polochon.ShowEpisode:
		return p.getEpisodeDetails(resource)
	default:
		return ErrInvalidType
	}
}
