package dm

import (
	"os"
	"path/filepath"
	"time"

	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/app/subapp"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/sirupsen/logrus"
)

// AppName is the application name
const AppName = "download_manager"

// DownloadManager represents the download manager
type DownloadManager struct {
	*subapp.Base

	library *library.Library
	config  *configuration.Config
}

// New returns a new download manager
func New(config *configuration.Config, library *library.Library) *DownloadManager {
	return &DownloadManager{
		Base:    subapp.NewBase(AppName),
		config:  config,
		library: library,
	}
}

// Run starts the download manager
func (dm *DownloadManager) Run(log *logrus.Entry) error {
	log = log.WithField("app", AppName)

	// Init the app
	dm.InitStart(log)

	log.Debug("download manager started")

	log.Debug("initial download manager launch")

	var err error
	dm.Wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err = errors.New("panic recovered").Fatal().AddContext(errors.Context{
					"sub_app": AppName,
				})
				dm.Stop(log)
			}

			dm.Wg.Done()
		}()
		dm.run(log)
	}()

	// Start the download manager
	dm.Wg.Add(1)
	go func() {
		defer dm.Wg.Done()
		dm.ticker(log)
	}()

	defer log.Debug("download manager stopped")

	dm.Wg.Wait()

	return err
}

func (dm *DownloadManager) ticker(log *logrus.Entry) {
	ticker := time.NewTicker(dm.config.DownloadManager.Timer)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			dm.run(log)
		case <-dm.Done:
			log.Debug("download manager timer stopped")
			return
		}
	}
}

func (dm *DownloadManager) run(log *logrus.Entry) {
	// Organise all the files in the files directory stuff
	// Start the fs notifier on this directory
	// Start the torrent list stuff
	torrents, err := dm.config.Downloader.Client.List()
	if err != nil {
		log.Errorf("error while getting torrent list: %q", err)
		return
	}

	for _, torrent := range torrents {
		if torrent.Status == nil {
			continue
		}

		tlog := log.WithField("torrent_name", torrent.Status.Name)
		if !torrent.Status.IsFinished {
			continue
		}

		video := torrent.Video()
		if video == nil {
			tlog.Debugf("torrent is not a video")
			dm.moveToWatcherDirectory(torrent, tlog)
			continue
		}

		file := dm.findVideoFile(torrent)
		if file == nil {
			tlog.Debugf("torrent video file not found")
			dm.moveToWatcherDirectory(torrent, tlog)
			continue
		}
		video.SetFile(*file)

		if file.IsSymlink() {
			if torrent.RatioReached(dm.config.DownloadManager.Ratio) {
				dm.cleanTorrent(torrent, tlog)
			}
			continue
		}

		metadata, err := file.GuessMetadata()
		if err != nil {
			tlog.Warnf("failed to guess metadata: %s", err.Error())
		}
		video.SetMetadata(metadata)

		// TODO: update the lib to handle this
		switch v := video.(type) {
		case *polochon.Movie:
			v.MovieConfig = dm.config.Movie
		case *polochon.ShowEpisode:
			v.ShowConfig = dm.config.Show
		default:
			dm.moveToWatcherDirectory(torrent, tlog)
			continue
		}

		// Get the video details
		if err := polochon.GetDetails(video, tlog); err != nil {
			errors.LogErrors(tlog, err)
			if errors.IsFatal(err) {
				dm.moveToWatcherDirectory(torrent, tlog)
				continue
			}
		}

		// Get the video subtitles
		if err := polochon.GetSubtitles(video, dm.config.SubtitleLanguages, tlog); err != nil {
			errors.LogErrors(tlog, err)
		}

		// Store the video
		if err := dm.library.Add(video, tlog); err != nil {
			errors.LogErrors(tlog, err)
			dm.moveToWatcherDirectory(torrent, tlog)
			continue
		}

		// Notify
		dm.Notify(video, tlog)

		tlog.Debugf("torrent organized")
	}
}

func (dm *DownloadManager) findVideoFile(torrent *polochon.Torrent) *polochon.File {
	for _, tPath := range torrent.Status.FilePaths {
		filePath := filepath.Join(dm.config.DownloadManager.Dir, tPath)
		file := polochon.NewFileWithConfig(filePath, dm.config.File)

		if !file.Exists() || file.IsExcluded() {
			continue
		}

		if file.IsVideo() {
			return file
		}
	}

	return nil
}

func (dm *DownloadManager) moveToWatcherDirectory(torrent *polochon.Torrent, log *logrus.Entry) {
	log.Infof("moving to the watcher directory")

	// Extract the top path of the directories and the path of the files
	fileMap := map[string]struct{}{}
	for _, p := range torrent.Status.FilePaths {
		top := p
		for filepath.Dir(top) != "." {
			top = filepath.Dir(top)
		}
		fileMap[top] = struct{}{}
	}

	for p := range fileMap {
		oldPath := filepath.Join(dm.config.DownloadManager.Dir, p)
		newPath := filepath.Join(dm.config.Watcher.Dir, p)
		log.Debugf("moving %s to %s", oldPath, newPath)
		if err := os.Rename(oldPath, newPath); err != nil {
			log.Errorf("error while moving torrent file: %s", err.Error())
		}
	}

	dm.cleanTorrent(torrent, log)
}

// Notify sends video to the notifiers
func (dm *DownloadManager) Notify(v polochon.Video, log *logrus.Entry) {
	log = log.WithField("function", "notify")
	for _, n := range dm.config.Notifiers {
		if err := n.Notify(v, log); err != nil {
			log.Warnf("failed to send a notification from notifier: %q: %q", n.Name(), err)
		}
	}
}
