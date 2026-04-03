package dm

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"

	polochon "github.com/odwrtw/polochon/lib"
)

func (dm *DownloadManager) cleanTorrent(torrent *polochon.Torrent) {
	if torrent.Status == nil {
		return
	}

	log := dm.log.With("torrent_name", torrent.Status.Name)

	// We remove the torrent
	log.Debug("removing torrent")
	err := dm.config.Downloader.Client.Remove(torrent)
	if err != nil {
		log.Error("got error when removing torrent", "error", err)
		return
	}

	// Going over all the files and remove only the allowed ones
	log.Debug("cleaning torrent files")
	for _, tPath := range torrent.Status.FilePaths {
		filePath := filepath.Join(dm.config.DownloadManager.Dir, tPath)
		file := polochon.NewFile(filePath)

		// Check extension
		ext := path.Ext(filePath)
		if !slices.Contains(dm.config.File.AllowedExtensionsToDelete, ext) {
			if !file.IsSymlink() {
				// Not allowed to delete these types of files
				log.Debug("protected extension", "extension", ext, "file_to_clean", filePath)
				continue
			} else {
				log.Debug("file is a symlink, delete it", "file", filePath)
			}
		}

		err := dm.remove(file.Path)
		if err != nil {
			log.Warn("got error while removing file", "error", err)
			continue
		}
	}

	// Need to check if we can delete the directory of the torrent
	err = dm.cleanDirectory(torrent)
	if err != nil {
		log.Warn("got error while deleting directory", "error", err)
	}
}

func (dm *DownloadManager) remove(filePath string) error {
	dm.log.Debug("deleting item", "path", filePath)
	return os.Remove(filePath)
}

func (dm *DownloadManager) cleanDirectory(torrent *polochon.Torrent) error {
	if torrent.Status == nil {
		return fmt.Errorf("missing torrent status")
	}

	log := dm.log.With("torrent_name", torrent.Status.Name)

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
		log.Debug("going higher", "dir", relDir)
	}

	// Get the full path
	directoryToClean := filepath.Join(dm.config.DownloadManager.Dir, relDir)
	log.Debug("try to clean and delete")

	ok, err := IsEmpty(directoryToClean)
	if err != nil {
		log.Warn("got error checking if directory is empty", "error", err)
		return err
	}
	if !ok {
		log.Debug("directory is not empty")
		return nil
	}

	log.Debug("everything is ready to delete the dir")

	// Delete the directory
	return dm.remove(directoryToClean)
}

// IsEmpty checks if a directory is empty
func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}
