package dm

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

func (dm *DownloadManager) cleanTorrent(torrent *polochon.Torrent, log *logrus.Entry) {
	if torrent.Status == nil {
		return
	}

	// We remove the torrent
	log.Debugf("removing torrent")
	err := dm.config.Downloader.Client.Remove(torrent)
	if err != nil {
		log.Errorf("got error when removing torrent : %q", err)
		return
	}

	// Going over all the files and remove only the allowed ones
	log.Debugf("cleaning torrent files")
	for _, tPath := range torrent.Status.FilePaths {
		filePath := filepath.Join(dm.config.DownloadManager.Dir, tPath)
		file := polochon.NewFile(filePath)

		// Check extension
		ext := path.Ext(filePath)
		if !stringInSlice(ext, dm.config.File.AllowedExtensionsToDelete) {
			if !file.IsSymlink() {
				// Not allowed to delete these types of files
				log.WithFields(logrus.Fields{
					"extension":     ext,
					"file_to_clean": filePath,
				}).Debug("protected extension")
				continue
			} else {
				log.Debugf("file %q is a symlink, delete it", filePath)
			}
		}

		err := dm.remove(file.Path, log)
		if err != nil {
			log.Warnf("got error while removing file %q", err)
			continue
		}
	}

	// Need to check if we can delete the directory of the torrent
	err = dm.cleanDirectory(torrent, log)
	if err != nil {
		log.Warnf("got error while deleting directory : %q", err)
	}
}

func (dm *DownloadManager) remove(filePath string, log *logrus.Entry) error {
	log.WithField("path", filePath).Debug("deleting item")
	return os.Remove(filePath)
}

func (dm *DownloadManager) cleanDirectory(torrent *polochon.Torrent, log *logrus.Entry) error {
	if torrent.Status == nil {
		return fmt.Errorf("missing torrent status")
	}

	if len(torrent.Status.FilePaths) == 0 {
		return fmt.Errorf("no torrent files to clean")
	}

	// Get the path of one of the file to guess the directory that needs to be
	// deleted
	torrentFilePath := torrent.Status.FilePaths[0]

	// Get the full path of the file
	filePath := filepath.Join(dm.config.DownloadManager.Dir, torrentFilePath)
	// Get the directory of the file
	directoryPath := filepath.Dir(filePath)
	// Ensure the path is clean
	directoryPath = filepath.Clean(directoryPath)
	// We don't want to clean the DownloadDir
	if directoryPath == dm.config.DownloadManager.Dir {
		log.Debug("in the download folder, no need to clean")
		return nil
	}

	// Get relative path of the directory to clean
	relDir, err := filepath.Rel(dm.config.DownloadManager.Dir, directoryPath)
	if err != nil {
		return err
	}

	// Get the first part of the directory to clean
	for filepath.Dir(relDir) != "." {
		relDir = filepath.Dir(relDir)
		log.Debugf("going higher : %s", relDir)
	}

	// Get the full path
	directoryToClean := filepath.Join(dm.config.DownloadManager.Dir, relDir)
	log.Debug("try to clean and delete")

	ok, err := IsEmpty(directoryToClean)
	if err != nil {
		log.Warnf("got error checking if directory is empty : %q", err)
		return err
	}
	if !ok {
		log.Debug("directory is not empty")
		return nil
	}

	log.Debug("everything is ready to delete the dir")

	// Delete the directory
	return dm.remove(directoryToClean, log)
}

// IsEmpty checks if a directory is empty
func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

// Helper to check if a string is included in a slice
func stringInSlice(s string, slice []string) bool {
	for _, e := range slice {
		if e == s {
			return true
		}
	}
	return false
}
