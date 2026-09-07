package library

import (
	"context"
	"os"
	"path/filepath"

	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

// ShowIDs returns the show ids, seasons and episodes
func (l *Library) ShowIDs() map[string]*index.Show {
	return l.showIndex.Index()
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

// DeleteShow deletes the whole show
func (l *Library) DeleteShow(id string) error {
	log := l.log.With("type", "show", "imdb_id", id)
	path, err := l.showIndex.ShowPath(id)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(path); err != nil {
		return err
	}

	log.Info("removing show")
	// Remove the show from the index
	show := &polochon.Show{ImdbID: id}
	return l.showIndex.RemoveShow(show)
}

// GetIndexedShow returns an indexed Show from its id
func (l *Library) GetIndexedShow(id string) (*index.Show, error) {
	s, err := l.showIndex.IndexedShow(id)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (l *Library) addShow(ep *polochon.ShowEpisode) error {
	dir := l.getShowDir(ep)
	nfoPath := l.showNFOPath(dir)
	nfoExists := exists(nfoPath)
	if nfoExists {
		s, err := l.newShowFromPath(nfoPath)
		if err == nil {
			ep.Show = s
			return nil
		}
		l.log.Warn("failed to read show NFO, repairing it", "path", nfoPath, "error", err)
	}

	s := ep.Show
	if s == nil {
		s = polochon.NewShowFromEpisode(ep)
		if err := polochon.GetDetails(context.Background(), s, l.log); err != nil {
			return err
		}
	}
	ep.Show = s

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
	if nfoExists {
		return nil
	}

	// Download show images
	if s.FanartURL == "" || s.BannerURL == "" || s.PosterURL == "" {
		return ErrMissingShowImageURL
	}

	// Download images
	for _, img := range []struct {
		url  string
		name string
	}{
		{
			url:  s.FanartURL,
			name: "fanart.jpg",
		},
		{
			url:  s.PosterURL,
			name: "poster.jpg",
		},
		{
			url:  s.BannerURL,
			name: "banner.jpg",
		},
	} {
		l.log.Debug("downloading " + img.name)
		savePath := filepath.Join(dir, img.name)
		if err := download(img.url, savePath); err != nil {
			return err
		}
	}

	return nil
}

// newShowFromPath returns a new Show from its path
func (l *Library) newShowFromPath(path string) (*polochon.Show, error) {
	s := polochon.NewShow(l.showConfig)
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
