package library

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetSubtitle returns the subtitle if it exists, nil otherwise
func (l *Library) GetSubtitle(v polochon.Video, lang polochon.Language) *polochon.Subtitle {
	sub := polochon.NewSubtitleFromVideo(v, lang)
	if !sub.Exists() {
		return nil
	}

	return sub
}

// UpdateSubtitles adds the subtitles to the video if the files are found
func (l *Library) UpdateSubtitles(v polochon.Video) {
	subs := []*polochon.Subtitle{}
	for _, lang := range l.SubtitleLanguages {
		if s := l.GetSubtitle(v, lang); s != nil {
			subs = append(subs, s)
		}
	}

	if len(subs) > 0 {
		v.SetSubtitles(subs)
	}
}

// SaveSubtitles saves the subtitles of a video
func (l *Library) SaveSubtitles(video polochon.Video, log *logrus.Entry) error {
	for _, s := range video.GetSubtitles() {
		if err := s.Save(); err != nil {
			return err
		}

		log.WithFields(logrus.Fields{
			"lang": string(s.Lang),
			"path": s.Path,
		}).Debugf("subtitle saved")
	}

	return nil
}
