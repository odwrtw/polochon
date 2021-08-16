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

// GetSubtitle gets the subtitles of a video in the given languages
func GetSubtitle(video Video, lang Language, log *logrus.Entry) (*Subtitle, error) {
	c := errors.NewCollector()

	var found *Subtitle

	// Ask all the subtitlers
	for _, subtitler := range video.GetSubtitlers() {
		l := log.WithFields(logrus.Fields{
			"subtitler": subtitler.Name(),
			"lang":      lang,
		})
		l.Debugf("searching subtitle")

		subtitle, err := subtitler.GetSubtitle(video, lang, l)
		if err != nil {
			c.Push(errors.Wrap(err).Ctx("subtitler", subtitler.Name()).Ctx("lang", lang))
			continue
		}

		found = subtitle
	}

	if c.HasErrors() {
		if c.IsFatal() {
			return nil, c
		}
		log.Warnf("got non fatal errors while getting subtitles: %s", c)
	}

	if found == nil {
		return nil, ErrNoSubtitleFound
	}

	idx := -1
	subtitles := video.GetSubtitles()
	for i, s := range subtitles {
		if s.Lang == lang {
			idx = i
			break
		}
	}

	// Add or replace the subtitle
	if idx >= 0 {
		subtitles[idx] = found
	} else {
		subtitles = append(subtitles, found)
	}

	video.SetSubtitles(subtitles)
	return found, nil
}
