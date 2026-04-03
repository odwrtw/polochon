package dm

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
)

// AppName is the application name
const AppName = "download_manager"

// DownloadManager represents the download manager
type DownloadManager struct {
	log     *slog.Logger
	library *library.Library
	config  *configuration.Config
}

// New returns a new download manager
func New(config *configuration.Config, library *library.Library, log *slog.Logger) *DownloadManager {
	return &DownloadManager{
		log:     log.With("app", AppName),
		config:  config,
		library: library,
	}
}

// Run starts the download manager
func (dm *DownloadManager) Run(ctx context.Context) error {
	dm.log.Debug("download manager started")
	dm.run(ctx)

	ticker := time.NewTicker(dm.config.DownloadManager.Timer)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dm.run(ctx)
		case <-ctx.Done():
			dm.log.Debug("download manager stopped")
			return nil
		}
	}
}

func (dm *DownloadManager) run(ctx context.Context) {
	torrents, err := dm.config.Downloader.Client.List()
	if err != nil {
		dm.log.Error("error while getting torrent list", "error", err)
		return
	}

	for _, torrent := range torrents {
		if torrent.Status == nil {
			continue
		}

		tlog := dm.log.With("torrent_name", torrent.Status.Name)
		if !torrent.Status.IsFinished {
			continue
		}

		video := torrent.Video()
		if video == nil {
			tlog.Debug("torrent is not a video")
			dm.moveToWatcherDirectory(torrent)
			continue
		}

		file := dm.findVideoFile(torrent)
		if file == nil {
			tlog.Debug("torrent video file not found")
			dm.moveToWatcherDirectory(torrent)
			continue
		}
		video.SetFile(*file)

		if file.IsSymlink() {
			if torrent.RatioReached(dm.config.DownloadManager.Ratio) {
				dm.cleanTorrent(torrent)
			}
			continue
		}

		metadata, err := file.GuessMetadata(tlog)
		if err != nil {
			tlog.Warn("failed to guess metadata", "error", err)
		}
		video.SetMetadata(metadata)

		// TODO: update the lib to handle this
		switch v := video.(type) {
		case *polochon.Movie:
			v.MovieConfig = dm.config.Movie
		case *polochon.ShowEpisode:
			v.ShowConfig = dm.config.Show
		default:
			dm.moveToWatcherDirectory(torrent)
			continue
		}

		// Get the video details
		if err := polochon.GetDetails(ctx, video, tlog); err != nil {
			if err != polochon.ErrGettingDetails {
				tlog.Error(err.Error())
			}

			dm.moveToWatcherDirectory(torrent)
			continue
		}

		// Get the video subtitles
		for _, lang := range dm.config.SubtitleLanguages {
			_, err := polochon.GetSubtitle(ctx, video, lang, tlog)
			if err != nil && err != polochon.ErrNoSubtitleFound {
				tlog.Error(err.Error())
			}
		}

		// Store the video
		if err := dm.library.Add(video); err != nil {
			tlog.Error(err.Error())
			dm.moveToWatcherDirectory(torrent)
			continue
		}

		// Notify
		dm.Notify(ctx, video)

		tlog.Debug("torrent organized")
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

func (dm *DownloadManager) moveToWatcherDirectory(torrent *polochon.Torrent) {
	log := dm.log.With("torrent_name", torrent.Status.Name)
	log.Info("moving to the watcher directory")

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
		log.Debug("moving file", "from", oldPath, "to", newPath)
		if err := library.MoveFile(oldPath, newPath); err != nil {
			log.Error("error while moving torrent file", "error", err)
		}
	}

	dm.cleanTorrent(torrent)
}

// Notify sends video to the notifiers
func (dm *DownloadManager) Notify(ctx context.Context, v polochon.Video) {
	log := dm.log.With("function", "notify")
	for _, n := range dm.config.Notifiers {
		if err := n.Notify(ctx, v); err != nil {
			log.Warn("failed to send a notification from notifier", "notifier", n.Name(), "error", err)
		}
	}
}
