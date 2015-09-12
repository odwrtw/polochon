package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

type cleaner struct {
	config *Config
	event  chan struct{}
	done   chan struct{}
	stop   chan struct{}
	errc   chan error
	wg     *sync.WaitGroup
	log    *logrus.Entry
}

// Errors
var (
	ErrProtectedFromCleaner = errors.New("file protected from cleaner")
)

func (c *cleaner) cleanDaemon() {
	c.log.Info("Starting cleaner")

	// Start the ticker
	go c.ticker()

	// Start the cleaner
	go c.cleaner()

	// Lauch the cleaner at startup
	go func() {
		c.log.Debug("initial cleaner launch")
		c.event <- struct{}{}
	}()
}

func (c *cleaner) ticker() {
	c.wg.Add(1)
	defer c.wg.Done()
	tick := time.Tick(c.config.Downloader.Cleaner.Timer)
	for {
		select {
		case <-tick:
			c.log.Debug("Cleaner timer triggered")
			c.event <- struct{}{}
		case <-c.done:
			return
		}
	}
}

func (c *cleaner) cleaner() {
	c.wg.Add(1)
	defer c.wg.Done()
	for {
		select {
		case <-c.event:
			c.log.Debug("Cleaner event !")
			c.cleanDoneVideos(c.log)
		case <-c.done:
			c.log.Debug("Cleaner done")
			return
		}
	}
}

func (c *cleaner) cleanDoneVideos(log *logrus.Entry) {
	list, err := c.config.Downloader.Client.List()
	if err != nil {
		log.Warnf("Error while getting torrent list", err)
		return
	}

	for _, t := range list {
		torrentInfos := t.Infos()

		log = log.WithField("torrentName", torrentInfos.Name)
		// Check if the file is ready to be cleaned
		isReady := c.isReadyToBeCleaned(t, log)
		if !isReady {
			log.Debug("torrent is not ready to be cleaned")
			continue
		}

		// We remove the torrent
		log.Debugf("Removing torrent")
		err := c.config.Downloader.Client.Remove(t)
		if err != nil {
			log.Warnf("Got error when removing torrent : %q", err)
			continue
		}

		log.Debug("Removing files")
		if err = c.clean(t, log); err != nil {
			log.Warnf("Failed to clean torrent files", err)
			continue
		}
	}
}

func (c *cleaner) isReadyToBeCleaned(d polochon.Downloadable, log *logrus.Entry) bool {
	tInfos := d.Infos()
	cLogger := log.WithField("torrentName", tInfos.Name)

	// First check that the torrent download is finished
	if !tInfos.IsFinished {
		cLogger.Debugf("torrent is not yet finished")
		return false
	}

	// Check that the ratio is reached
	if tInfos.Ratio < c.config.Downloader.Cleaner.Ratio {
		cLogger.Debugf("ratio is not reached (%f / %f)", tInfos.Ratio, c.config.Downloader.Cleaner.Ratio)
		return false
	}

	return true
}

func (c *cleaner) clean(d polochon.Downloadable, log *logrus.Entry) error {
	tInfos := d.Infos()

	// Going over all the files and remove only the allowed ones
	for _, tPath := range tInfos.FilePaths {
		filePath := filepath.Join(c.config.Watcher.Dir, tPath)
		file := polochon.NewFile(filePath)

		// Check extention
		ext := path.Ext(filePath)
		if !stringInSlice(ext, c.config.File.AllowedExtentionsToDelete) {
			if !file.IsSymlink() {
				// Not allowed to delete these types of files
				log.WithFields(logrus.Fields{
					"extension":   ext,
					"fileToClean": filePath,
				}).Debug("Protected extention!")
				continue
			} else {
				log.Debugf("File %q is a sym link, we can delete it", filePath)
			}
		}

		// If it's a symlink, delete the file as it has already been organized
		err := c.deleteFile(file.Path, log)
		if err != nil {
			log.Warnf("Got error while removing file %q", err)
			continue
		}
	}

	// Need to check if we can delete the directory of the torrent
	err := c.cleanDirectory(tInfos, log)
	if err != nil {
		log.Warnf("Got error while deleting directory : %q", err)
		return err
	}

	return nil
}

func (c *cleaner) deleteFile(filePath string, log *logrus.Entry) error {
	// Delete the file
	deletePath := filepath.Join(c.config.Downloader.Cleaner.TrashDir, path.Base(filePath))

	log.Debug("Deleting file ... ")
	log.Debugf("%s ---> %s", filePath, deletePath)

	return os.Rename(filePath, deletePath)
}

func (c *cleaner) deleteDirectory(directoryPath string, log *logrus.Entry) error {
	// Delete the directory
	deletePath := filepath.Join(c.config.Downloader.Cleaner.TrashDir, path.Base(directoryPath))

	log.Debug("Deleting folder ... ")
	log.Debugf("%s ---> %s", directoryPath, deletePath)

	return os.Rename(directoryPath, deletePath)
}

func (c *cleaner) cleanDirectory(tInfos *polochon.DownloadableInfos, log *logrus.Entry) error {
	if len(tInfos.FilePaths) == 0 {
		return fmt.Errorf("no torrent files to clean")
	}

	// Get the path of one of the file to guess the directory that needs to be
	// deleted
	torrentFilePath := tInfos.FilePaths[0]

	// Get the full path of the file
	filePath := filepath.Join(c.config.Watcher.Dir, torrentFilePath)
	// Get the directory of the file
	directoryPath := filepath.Dir(filePath)
	// Ensure the path is clean
	directoryPath = filepath.Clean(directoryPath)
	// We don't want to clean the DownloadDir
	if directoryPath == c.config.Downloader.DownloadDir {
		log.Debug("We're in the watching folder, no need to clean")
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
		log.Debugf("Going higher : %s", relDir)
	}

	// Get the full path
	directoryToClean := filepath.Join(c.config.Downloader.DownloadDir, relDir)
	log.Debug("Try to clean and delete")

	ok, err := IsEmpty(directoryToClean)
	if err != nil {
		log.Warnf("Got error checking if directory is empty : %q", err)
		return err
	}
	if !ok {
		log.Debug("Directory is not empty")
		return nil
	}

	log.Debug("Everything is ready to delete the dir")

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
