package bsplayer

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/agnivade/levenshtein"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

const moduleName = "bsplayer"

var _ polochon.Subtitler = (*Client)(nil)

var ErrNotAVideo = errors.New("bsplayer: not a video")

func init() {
	polochon.RegisterModule(&Client{})
}

// Client represents the bsplayer API client.
type Client struct {
}

// Init implements the polochon.Module interface.
func (c *Client) Init(_ []byte) error {
	return nil
}

// Name implements the polochon.Module interface.
func (c *Client) Name() string {
	return moduleName
}

// Status implements the polochon.Module interface.
func (c *Client) Status() (polochon.ModuleStatus, error) {
	qp := &queryParams{
		imdbID: "133093",
		size:   "1991716652",
		hash:   "6513e3c7b21e645c",
		lang:   "eng",
	}

	subs, err := search(qp)
	if err != nil || len(subs) == 0 {
		return polochon.StatusFail, err
	}

	return polochon.StatusOK, nil
}

// ListSubtitles implements the polochon.Subtitler interface.
func (c *Client) ListSubtitles(i any, lang polochon.Language, _ *logrus.Entry) ([]*polochon.SubtitleEntry, error) {
	var qp *queryParams
	var err error

	switch resource := i.(type) {
	case *polochon.Movie:
		qp, err = newQuery(resource.ImdbID, lang, resource.GetFile())
	case *polochon.ShowEpisode:
		qp, err = newQuery(resource.ShowImdbID, lang, resource.GetFile())
	default:
		return nil, ErrNotAVideo
	}

	if err != nil {
		return nil, err
	}

	subs, err := search(qp)
	if err != nil {
		return nil, err
	}

	if len(subs) == 0 {
		return nil, polochon.ErrNoSubtitleFound
	}

	entries := make([]*polochon.SubtitleEntry, 0, len(subs))
	for _, s := range subs {
		id, err := encodeDownloadID(s.URL)
		if err != nil {
			continue
		}
		entries = append(entries, &polochon.SubtitleEntry{
			Language:    lang,
			Description: fmt.Sprintf("%s (Rating: %s)", s.Name, s.Rating),
			ID:          id,
		})
	}
	return entries, nil
}

// DownloadSubtitle implements the polochon.Subtitler interface.
func (c *Client) DownloadSubtitle(i any, entry *polochon.SubtitleEntry, _ *logrus.Entry) (*polochon.Subtitle, error) {
	video, ok := i.(polochon.Video)
	if !ok {
		return nil, ErrNotAVideo
	}

	downloadURL, err := decodeDownloadURL(entry.ID)
	if err != nil {
		return nil, err
	}
	rc, err := fetch(downloadURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	s := polochon.NewSubtitleFromVideo(video, entry.Language)
	s.Data, err = io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GetSubtitle implements the polochon.Subtitler interface.
func (c *Client) GetSubtitle(i any, lang polochon.Language, _ *logrus.Entry) (*polochon.Subtitle, error) {
	var qp *queryParams
	var err error

	switch resource := i.(type) {
	case *polochon.Movie:
		qp, err = newQuery(resource.ImdbID, lang, resource.GetFile())
	case *polochon.ShowEpisode:
		qp, err = newQuery(resource.ShowImdbID, lang, resource.GetFile())
	default:
		return nil, ErrNotAVideo
	}

	if err != nil {
		return nil, err
	}

	subs, err := search(qp)
	if err != nil {
		return nil, err
	}

	video, ok := i.(polochon.Video)
	if !ok {
		return nil, ErrNotAVideo
	}

	var selected *subtitle
	minScore := 1000

	release := filepath.Base(video.GetFile().PathWithoutExt())
	for _, sub := range subs {
		dist := levenshtein.ComputeDistance(release, sub.Name)
		if dist < minScore {
			selected = sub
			minScore = dist
		}
	}

	if selected == nil {
		return nil, polochon.ErrNoSubtitleFound
	}

	rc, err := fetch(selected.URL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	s := polochon.NewSubtitleFromVideo(video, lang)
	s.Data, err = io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return s, nil
}
