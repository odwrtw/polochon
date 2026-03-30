package polochon

import (
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
)

// ErrNoSubtitleFound trigger when no subtitle found
var ErrNoSubtitleFound = errors.New("polochon: no subtitle found")

// SubtitleEntry represents a subtitle available for download (without the data itself)
type SubtitleEntry struct {
	Language    Language `json:"language"`
	Embedded    bool     `json:"embedded"`
	Source      string   `json:"source"`
	ID          string   `json:"id"`
	Description string   `json:"description"`
}

// ListSubtitles returns all available subtitles for a video in the given language
// across all configured subtitlers, without downloading the subtitle data.
func ListSubtitles(video Video, lang Language, log *logrus.Entry) ([]*SubtitleEntry, error) {
	resultCh := make(chan []*SubtitleEntry)
	done := make(chan struct{})

	var result []*SubtitleEntry
	go func() {
		defer close(done)
		for entries := range resultCh {
			result = append(result, entries...)
		}
	}()

	var wg sync.WaitGroup
	for _, subtitler := range video.GetSubtitlers() {
		wg.Go(func() {
			l := log.WithFields(logrus.Fields{
				"subtitler": subtitler.Name(),
				"lang":      lang,
			})
			l.Debug("listing subtitles")

			entries, err := subtitler.ListSubtitles(video, lang, l)
			if err != nil {
				switch err {
				case ErrNotAvailable:
					// nothing to log
				case ErrNoSubtitleFound:
					l.Debug("no subtitles found")
				default:
					l.Warn(err)
				}
				return
			}

			for _, e := range entries {
				e.Source = subtitler.Name()
			}
			resultCh <- entries
		})
	}

	wg.Wait()
	close(resultCh)
	<-done

	if len(result) == 0 {
		return nil, ErrNoSubtitleFound
	}

	return result, nil
}

// GetSubtitle gets the subtitles of a video in the given languages
func GetSubtitle(video Video, lang Language, log *logrus.Entry) (*Subtitle, error) {
	var found *Subtitle

	// Ask all the subtitlers
	for _, subtitler := range video.GetSubtitlers() {
		l := log.WithFields(logrus.Fields{
			"subtitler": subtitler.Name(),
			"lang":      lang,
		})
		l.Debug("searching subtitle")

		subtitle, err := subtitler.GetSubtitle(video, lang, l)
		if err != nil {
			switch err {
			case ErrNotAvailable:
				// nothing to log
			case ErrNoSubtitleFound:
				l.Debug("no subtitle found")
			default:
				l.Warn(err)
			}
			continue
		}

		if subtitle != nil {
			found = subtitle
			break
		}
	}

	if found == nil {
		log.WithField("lang", lang).Debug("all subtitlers failed to find a subtitle")
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
