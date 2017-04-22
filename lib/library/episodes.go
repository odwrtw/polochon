package library

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	polochon "github.com/odwrtw/polochon/lib"
)

// HasShowEpisode returns true if the show is in the store
func (l *Library) HasShowEpisode(imdbID string, season, episode int) (bool, error) {
	return l.showIndex.HasEpisode(imdbID, season, episode)
}

// AddShowEpisode adds an episode to the store
func (l *Library) AddShowEpisode(ep *polochon.ShowEpisode, log *logrus.Entry) error {
	if ep.Path == "" {
		return ErrMissingShowEpisodeFilePath
	}

	ok, err := l.HasShowEpisode(ep.ShowImdbID, ep.Season, ep.Episode)
	if err != nil {
		return err
	}
	if ok {
		// Get the old episode from the index
		oldEpisode, err := l.GetEpisode(ep.ShowImdbID, ep.Season, ep.Episode)
		if err != nil {
			return err
		}

		if err := l.DeleteShowEpisode(oldEpisode, log); err != nil {
			return err
		}
	}

	// Add the show
	if err := l.addShow(ep, log); err != nil {
		return err
	}

	// Create show season dir if necessary
	seasonDir := l.getSeasonDir(ep)
	if !exists(seasonDir) {
		if err := os.Mkdir(seasonDir, os.ModePerm); err != nil {
			return err
		}
	}

	// Move the file
	// If the show episode already in the right dir there is nothing to do
	if path.Dir(ep.Path) == seasonDir {
		log.Debug("show episode already in the destination folder")
		return nil
	}

	// Save the old path
	oldPath := ep.Path

	// Move the episode into the folder
	newPath := filepath.Join(seasonDir, path.Base(ep.Path))
	log.Debugf("Moving episode to folder Old path: %q, New path: %q", ep.Path, newPath)
	if err := os.Rename(ep.Path, newPath); err != nil {
		return err
	}

	// Set the new episode path
	ep.Path = newPath

	// Create a symlink between the new and the old location
	if err := os.Symlink(ep.Path, oldPath); err != nil {
		log.Warnf("Error while making symlink between %s and %s : %+v", oldPath, ep.Path, err)
	}

	// Create show NFO if necessary
	if err := writeNFOFile(ep.NfoPath(), ep); err != nil {
		return err
	}

	return l.showIndex.Add(ep)
}

// DeleteShowEpisode will delete the showEpisode
func (l *Library) DeleteShowEpisode(se *polochon.ShowEpisode, log *logrus.Entry) error {
	// Delete the episode
	log.Infof("Removing ShowEpisode %q", se.Path)
	// Remove the episode
	if err := os.RemoveAll(se.Path); err != nil {
		return err
	}
	pathWithoutExt := se.PathWithoutExt()
	// Remove also the .nfo and .srt files
	for _, ext := range []string{"nfo", "srt"} {
		fileToDelete := fmt.Sprintf("%s.%s", pathWithoutExt, ext)
		log.Debugf("Removing %q", fileToDelete)
		// Remove file
		if err := os.RemoveAll(fileToDelete); err != nil {
			return err
		}
	}

	// Remove the episode from the index
	if err := l.showIndex.RemoveEpisode(se, log); err != nil {
		return err
	}

	// Season is empty, delete the whole season
	ok, err := l.showIndex.IsSeasonEmpty(se.ShowImdbID, se.Season)
	if err != nil {
		return err
	}
	if ok {
		// Delete the whole season
		if err := l.DeleteSeason(se.ShowImdbID, se.Season, log); err != nil {
			return err
		}
	}

	return nil
}

// GetEpisode returns an episode if present in the index
func (l *Library) GetEpisode(imdbID string, season, episode int) (*polochon.ShowEpisode, error) {
	path, err := l.showIndex.EpisodePath(imdbID, season, episode)
	if err != nil {
		return nil, err
	}
	return l.newEpisodeFromPath(path)
}

// NewShowEpisodeFromPath returns a new ShowEpisode from its path
func (l *Library) newEpisodeFromPath(path string) (*polochon.ShowEpisode, error) {
	file := polochon.NewFile(path)
	se := polochon.NewShowEpisodeFromFile(l.showConfig, *file)

	if err := readNFOFile(file.NfoPath(), se); err != nil {
		return nil, err
	}

	return se, nil
}
