package cleaner

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/app/subapp"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/sirupsen/logrus"
)

// AppName is the application name
const AppName = "cleaner"

// Cleaner represents a cleaner
type Cleaner struct {
	*subapp.Base

	config *configuration.Config
	event  chan struct{}
}

// New returns a new cleaner
func New(config *configuration.Config) *Cleaner {
	return &Cleaner{
		Base:   subapp.NewBase(AppName),
		config: config,
	}
}

// Run starts the cleaner
func (c *Cleaner) Run(log *logrus.Entry) error {
	log = log.WithField("app", AppName)

	// Init the app
	c.InitStart(log)

	c.event = make(chan struct{}, 1)

	log.Debug("cleaner started")

	log.Debug("initial cleaner launch")
	c.event <- struct{}{}

	// Start the ticker
	c.Wg.Add(1)
	go func() {
		defer c.Wg.Done()
		c.ticker(log)
	}()

	// Start the cleaner
	var err error
	c.Wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err = errors.New("panic recovered").Fatal().AddContext(errors.Context{
					"sub_app": AppName,
				})
				c.Stop(log)
			}

			c.Wg.Done()
		}()
		c.cleaner(log)
	}()

	defer log.Debug("cleaner stopped")

	c.Wg.Wait()

	return err
}

func (c *Cleaner) ticker(log *logrus.Entry) {
	ticker := time.NewTicker(c.config.Downloader.Cleaner.Timer)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Debug("cleaner timer triggered")
			c.event <- struct{}{}
		case <-c.Done:
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
		case <-c.Done:
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

		// Check extension
		ext := path.Ext(filePath)
		if !stringInSlice(ext, c.config.File.AllowedExtentionsToDelete) {
			if !file.IsSymlink() {
				// Not allowed to delete these types of files
				log.WithFields(logrus.Fields{
					"extension":     ext,
					"file_to_clean": filePath,
				}).Debug("protected extension")
				continue
			} else {
				log.Debugf("file %q is a sym link, delete it", filePath)
			}
		}

		// If it's a symlink, delete the file as it has already been organized
		err := c.remove(file.Path, log)
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

func (c *Cleaner) remove(filePath string, log *logrus.Entry) error {
	log.WithField("path", filePath).Debug("deleting item")
	return os.Remove(filePath)
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
	if directoryPath == c.config.Watcher.Dir {
		log.Debug("in the watching folder, no need to clean")
		return nil
	}

	// Get relative path of the directory to clean
	relDir, err := filepath.Rel(c.config.Watcher.Dir, directoryPath)
	if err != nil {
		return err
	}

	// Get the first part of the directory to clean
	for filepath.Dir(relDir) != "." {
		relDir = filepath.Dir(relDir)
		log.Debugf("going higher : %s", relDir)
	}

	// Get the full path
	directoryToClean := filepath.Join(c.config.Watcher.Dir, relDir)
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
	return c.remove(directoryToClean, log)
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
