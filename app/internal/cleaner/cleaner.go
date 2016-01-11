package cleaner

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/app/internal/configuration"
	"github.com/odwrtw/polochon/lib"
)

// AppName is the application name
const AppName = "cleaner"

// Cleaner represents a cleaner
type Cleaner struct {
	config *configuration.Config
	event  chan struct{}
	done   chan struct{}
}

// New returns a new cleaner
func New(config *configuration.Config) *Cleaner {
	return &Cleaner{
		config: config,
		done:   make(chan struct{}),
		event:  make(chan struct{}),
	}
}

// Name returns the name of the app
func (c *Cleaner) Name() string {
	return AppName
}

// Run starts the cleaner
func (c *Cleaner) Run(log *logrus.Entry) error {
	log = log.WithField("app", AppName)

	log.Debug("cleaner started")

	var wg sync.WaitGroup

	// Start the ticker
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.ticker(log)
	}()

	// Start the cleaner
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.cleaner(log)
	}()

	// Lauch the cleaner at startup
	go func() {
		log.Debug("initial cleaner launch")
		c.event <- struct{}{}
	}()

	wg.Wait()

	log.Debug("cleaner stopped")

	return nil
}

// Stop stops the cleaner
func (c *Cleaner) Stop(log *logrus.Entry) {
	close(c.done)
}

func (c *Cleaner) ticker(log *logrus.Entry) {
	tick := time.Tick(c.config.Downloader.Cleaner.Timer)
	for {
		select {
		case <-tick:
			log.Debug("cleaner timer triggered")
			c.event <- struct{}{}
		case <-c.done:
			log.Debug("cleaner timer stopped")
			return
		}
	}
}

func (c *Cleaner) cleaner(log *logrus.Entry) {
	for {
		select {
		case <-c.event:
			log.Debug("cleaner event")
			c.cleanDoneVideos(log)
		case <-c.done:
			log.Debug("cleaner done handling events")
			return
		}
	}
}

func (c *Cleaner) cleanDoneVideos(log *logrus.Entry) {
	list, err := c.config.Downloader.Client.List()
	if err != nil {
		log.Errorf("error while getting torrent list: %q", err)
		return
	}

	for _, t := range list {
		torrentInfos := t.Infos()

		log = log.WithField("torrent_name", torrentInfos.Name)

		// Check if the file is ready to be cleaned
		isReady := c.isReadyToBeCleaned(t, log)
		if !isReady {
			log.Debug("torrent is not ready to be cleaned")
			continue
		}

		// We remove the torrent
		log.Debugf("removing torrent")
		err := c.config.Downloader.Client.Remove(t)
		if err != nil {
			log.Errorf("got error when removing torrent : %q", err)
			continue
		}

		log.Debug("removing files")
		if err = c.clean(t, log); err != nil {
			log.Errorf("failed to clean torrent files: %q", err)
			continue
		}
	}
}

func (c *Cleaner) isReadyToBeCleaned(d polochon.Downloadable, log *logrus.Entry) bool {
	torrent := d.Infos()
	log = log.WithField("torrent_name", torrent.Name)

	// First check that the torrent download is finished
	if !torrent.IsFinished {
		log.Debugf("torrent is not yet finished")
		return false
	}

	// Check that the ratio is reached
	if torrent.Ratio < c.config.Downloader.Cleaner.Ratio {
		log.Debugf("ratio is not reached (%.02f / %.02f)", torrent.Ratio, c.config.Downloader.Cleaner.Ratio)
		return false
	}

	return true
}

func (c *Cleaner) clean(d polochon.Downloadable, log *logrus.Entry) error {
	torrent := d.Infos()

	// Going over all the files and remove only the allowed ones
	for _, tPath := range torrent.FilePaths {
		filePath := filepath.Join(c.config.Watcher.Dir, tPath)
		file := polochon.NewFile(filePath)

		// Check extention
		ext := path.Ext(filePath)
		if !stringInSlice(ext, c.config.File.AllowedExtentionsToDelete) {
			if !file.IsSymlink() {
				// Not allowed to delete these types of files
				log.WithFields(logrus.Fields{
					"extension":     ext,
					"file_to_clean": filePath,
				}).Debug("protected extention")
				continue
			} else {
				log.Debugf("file %q is a sym link, delete it", filePath)
			}
		}

		// If it's a symlink, delete the file as it has already been organized
		err := c.deleteFile(file.Path, log)
		if err != nil {
			log.Warnf("got error while removing file %q", err)
			continue
		}
	}

	// Need to check if we can delete the directory of the torrent
	err := c.cleanDirectory(torrent, log)
	if err != nil {
		log.Warnf("got error while deleting directory : %q", err)
		return err
	}

	return nil
}

func (c *Cleaner) deleteFile(filePath string, log *logrus.Entry) error {
	// Delete the file
	deletePath := filepath.Join(c.config.Downloader.Cleaner.TrashDir, path.Base(filePath))

	log.WithFields(logrus.Fields{
		"old_path": filePath,
		"new_path": deletePath,
	}).Debug("deleting file")

	return os.Rename(filePath, deletePath)
}

func (c *Cleaner) deleteDirectory(directoryPath string, log *logrus.Entry) error {
	// Delete the directory
	deletePath := filepath.Join(c.config.Downloader.Cleaner.TrashDir, path.Base(directoryPath))

	log.WithFields(logrus.Fields{
		"old_path": directoryPath,
		"new_path": deletePath,
	}).Debug("deleting folder")

	return os.Rename(directoryPath, deletePath)
}

func (c *Cleaner) cleanDirectory(torrent *polochon.DownloadableInfos, log *logrus.Entry) error {
	if len(torrent.FilePaths) == 0 {
		return fmt.Errorf("no torrent files to clean")
	}

	// Get the path of one of the file to guess the directory that needs to be
	// deleted
	torrentFilePath := torrent.FilePaths[0]

	// Get the full path of the file
	filePath := filepath.Join(c.config.Watcher.Dir, torrentFilePath)
	// Get the directory of the file
	directoryPath := filepath.Dir(filePath)
	// Ensure the path is clean
	directoryPath = filepath.Clean(directoryPath)
	// We don't want to clean the DownloadDir
	if directoryPath == c.config.Downloader.DownloadDir {
		log.Debug("in the watching folder, no need to clean")
		return nil
	}

	// Get relative path of the directory to clean
	relDir, err := filepath.Rel(c.config.Downloader.DownloadDir, directoryPath)
	if err != nil {
		return err
	}

	// Get the first part of the directory to clean
	for filepath.Dir(relDir) != "." {
		relDir = filepath.Dir(relDir)
		log.Debugf("going higher : %s", relDir)
	}

	// Get the full path
	directoryToClean := filepath.Join(c.config.Downloader.DownloadDir, relDir)
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
	if err = c.deleteDirectory(directoryToClean, log); err != nil {
		return err
	}

	return nil
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