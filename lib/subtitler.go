package polochon

import (
	errors "github.com/odwrtw/errors"
	"github.com/sirupsen/logrus"
)

// ErrNoSubtitleFound trigger when no subtitle found
var ErrNoSubtitleFound = errors.New("polochon: no subtitle found")

// Subtitler all subtitler must implement it
type Subtitler interface {
	Module
	GetSubtitle(interface{}, Language, *logrus.Entry) (*Subtitle, error)
}

// Subtitlable represents a ressource which can be subtitled
type Subtitlable interface {
	SubtitlePath(Language) string
	GetSubtitlers() []Subtitler
}

// GetSubtitles gets the subtitles of a video in the given languages
func GetSubtitles(video Video, languages []Language, log *logrus.Entry) error {
	c := errors.NewCollector()

	subtitles := []*Subtitle{}

	// We're going to ask subtitles in each language for each subtitles
	for _, lang := range languages {
		subtitlerLog := log.WithField("lang", lang)
		// Ask all the subtitlers
		for _, subtitler := range video.GetSubtitlers() {
			subtitlerLog = subtitlerLog.WithField("subtitler", subtitler.Name())
			subtitle, err := subtitler.GetSubtitle(video, lang, subtitlerLog)
			if err != nil {
				// If there was no errors, add the subtitle to the map of
				// subtitles
				c.Push(errors.Wrap(err).Ctx("Subtitler", subtitler.Name()).Ctx("lang", lang))
				continue
			}

			subtitles = append(subtitles, subtitle)
			break
		}
	}

	if c.HasErrors() {
		if c.IsFatal() {
			return c
		}
		log.Warnf("Got non fatal errors while getting subtitles: %s", c)
	}

	if len(subtitles) != 0 {
		video.SetSubtitles(subtitles)
	}

	return nil
}
