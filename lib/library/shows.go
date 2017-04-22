package library

import (
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

// ShowIDs returns the show ids, seasons and episodes
func (l *Library) ShowIDs() map[string]index.IndexedShow {
	return l.showIndex.IDs()
}

// GetShow returns a Show from its id
func (l *Library) GetShow(id string) (*polochon.Show, error) {
	path, err := l.showIndex.ShowPath(id)
	if err != nil {
		return nil, err
	}
	nfoPath := l.showNFOPath(path)

	s := polochon.NewShow(l.showConfig)
	if err := readNFOFile(nfoPath, s); err != nil {
		return nil, err
	}

	return s, nil
}

// GetIndexedShow returns an indexed Show from its id
func (l *Library) GetIndexedShow(id string) (index.IndexedShow, error) {
	s, err := l.showIndex.IndexedShow(id)
	if err != nil {
		return index.IndexedShow{}, err
	}

	return s, nil
}

func (l *Library) addShow(ep *polochon.ShowEpisode, log *logrus.Entry) error {
	dir := l.getShowDir(ep)
	nfoPath := l.showNFOPath(dir)
	if exists(nfoPath) {
		return nil
	}

	s := ep.Show
	if s == nil {
		s = polochon.NewShowFromEpisode(ep)
		if err := s.GetDetails(log); err != nil {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				return err
			}
		}
	}

	// Create show dir if necessary
	if !exists(dir) {
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			return err
		}
	}

	// Write NFO into the file
	if err := writeNFOFile(nfoPath, s); err != nil {
		return err
	}

	// Download show images
	if s.Fanart == "" || s.Banner == "" || s.Poster == "" {
		return ErrMissingShowImageURL
	}

	// Download images
	for _, img := range []struct {
		url  string
		name string
	}{
		{
			url:  s.Fanart,
			name: "fanart.jpg",
		},
		{
			url:  s.Poster,
			name: "poster.jpg",
		},
		{
			url:  s.Banner,
			name: "banner.jpg",
		},
	} {
		savePath := filepath.Join(dir, img.name)
		if err := download(img.url, savePath); err != nil {
			return err
		}
	}

	return nil
}

// newShowFromPath returns a new Show from its path
func (l *Library) newShowFromPath(path string) (*polochon.Show, error) {
	s := &polochon.Show{}
	if err := readNFOFile(path, s); err != nil {
		return nil, err
	}

	return s, nil
}

func (l *Library) getShowDir(ep *polochon.ShowEpisode) string {
	return filepath.Join(l.ShowDir, ep.ShowTitle)
}

func (l *Library) showNFOPath(showDir string) string {
	return filepath.Join(showDir, "tvshow.nfo")
}
