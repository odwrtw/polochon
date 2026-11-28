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

		if torrent.Type == polochon.TypeSeason {
			dm.organizeSeasonPack(ctx, tlog, torrent)
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

		if file.IsSymlink() {
			if torrent.RatioReached(dm.config.DownloadManager.Ratio) {
				dm.cleanTorrent(torrent)
			}
			continue
		}

		if !dm.organizeVideo(ctx, tlog, video, file) {
			dm.moveToWatcherDirectory(torrent)
		}
	}
}

// organizeSeasonPack organizes each episode of a season pack using the show
// and season from the torrent labels. Organized episodes are left as symlinks
// to keep seeding, so a pack is retried until every episode is organized.
func (dm *DownloadManager) organizeSeasonPack(ctx context.Context, log *slog.Logger, torrent *polochon.Torrent) {
	if dm.findVideoFile(torrent) == nil {
		dm.moveToWatcherDirectory(torrent)
		return
	}

	organized := true
	for _, p := range torrent.Status.FilePaths {
		file := polochon.NewFileWithConfig(filepath.Join(dm.config.DownloadManager.Dir, p), dm.config.File)
		if !file.Exists() || !file.IsVideo() || file.IsExcluded() || file.IsSymlink() {
			continue
		}
		organized = false

		guess, err := file.Guess(dm.config.Movie, dm.config.Show, log)
		e, ok := guess.(*polochon.ShowEpisode)
		if err != nil || !ok || e.Season != torrent.Season || e.Episode == 0 {
			log.Warn("season pack file is not an episode of the season", "file", p)
			dm.moveToWatcherDirectory(torrent)
			return
		}

		// Build the episode from the labels like single episode torrents
		label := *torrent
		label.Type = polochon.TypeEpisode
		label.Episode = e.Episode
		if !dm.organizeVideo(ctx, log, label.Video(), file) {
			dm.moveToWatcherDirectory(torrent)
			return
		}
	}

	if organized && torrent.RatioReached(dm.config.DownloadManager.Ratio) {
		dm.cleanTorrent(torrent)
	}
}

// organizeVideo adds a downloaded video to the library, it returns false if
// the video could not be organized
func (dm *DownloadManager) organizeVideo(ctx context.Context, log *slog.Logger, video polochon.Video, file *polochon.File) bool {
	video.SetFile(*file)

	metadata, err := file.GuessMetadata(log)
	if err != nil {
		log.Warn("failed to guess metadata", "error", err)
	}
	video.SetMetadata(metadata)

	// TODO: update the lib to handle this
	switch v := video.(type) {
	case *polochon.Movie:
		v.MovieConfig = dm.config.Movie
	case *polochon.ShowEpisode:
		v.ShowConfig = dm.config.Show
	default:
		return false
	}

	// Get the video details
	if err := polochon.GetDetails(ctx, video, log); err != nil {
		if err != polochon.ErrGettingDetails {
			log.Error(err.Error())
		}
		return false
	}

	// Get the video subtitles
	for _, lang := range dm.config.SubtitleLanguages {
		_, err := polochon.GetSubtitle(ctx, video, lang, log)
		if err != nil && err != polochon.ErrNoSubtitleFound {
			log.Error(err.Error())
		}
	}

	// Store the video
	if err := dm.library.Add(video); err != nil {
		log.Error(err.Error())
		return false
	}

	// Notify
	dm.Notify(ctx, video)

	log.Debug("torrent organized")
	return true
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
		// Organized files are symlinks to the library, drop them
		if file := polochon.NewFile(filepath.Join(dm.config.DownloadManager.Dir, p)); file.IsSymlink() {
			if err := dm.remove(file.Path); err != nil {
				log.Warn("error while removing symlink", "error", err)
			}
		}

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
