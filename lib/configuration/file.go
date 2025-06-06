package configuration

import (
	"errors"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/robfig/cron/v3"
)

type configFile struct {
	modulesParams *ModulesParams

	Logs      Logger          `yaml:"logs"`
	Organizer OrganizerConfig `yaml:"organizer"`

	Watcher struct {
		ModuleLoader `yaml:",inline"`
		Dir          string `yaml:"dir"`
	} `yaml:"watcher"`

	Downloader struct {
		ModuleLoader    `yaml:",inline"`
		LaunchAtStartup bool   `yaml:"launch_at_startup"`
		Enabled         bool   `yaml:"enabled"`
		Schedule        string `yaml:"schedule"`
	} `yaml:"downloader"`

	DownloadManager DownloadManagerConfig `yaml:"download_manager"`

	HTTPServer HTTPServer `yaml:"http_server"`

	Video struct {
		ModuleLoader              `yaml:",inline"`
		ExcludeFileContaining     []string            `yaml:"exclude_file_containing"`
		VideoExtensions           []string            `yaml:"allowed_file_extensions"`
		AllowedExtensionsToDelete []string            `yaml:"allowed_file_extensions_to_delete"`
		SubtitleLanguages         []polochon.Language `yaml:"subtitle_languages"`
	} `yaml:"video"`

	Show struct {
		ModuleLoader `yaml:",inline"`
		Dir          string `yaml:"dir"`
	} `yaml:"show"`

	Movie struct {
		ModuleLoader `yaml:",inline"`
		Dir          string `yaml:"dir"`
	} `yaml:"movie"`

	Wishlist struct {
		ModuleLoader          `yaml:",inline"`
		ShowDefaultQualities  []polochon.Quality `yaml:"show_default_qualities"`
		MovieDefaultQualities []polochon.Quality `yaml:"movie_default_qualities"`
	} `yaml:"wishlist"`
}

func loadConfig(cf *configFile, conf *Config) error {
	// Load the configs in the module loaders
	for _, ml := range []*ModuleLoader{
		&cf.Downloader.ModuleLoader,
		&cf.Movie.ModuleLoader,
		&cf.Show.ModuleLoader,
		&cf.Video.ModuleLoader,
		&cf.Watcher.ModuleLoader,
		&cf.Wishlist.ModuleLoader,
	} {
		ml.modulesParams = cf.modulesParams
		if err := ml.load(); err != nil {
			return err
		}
	}

	var schedule cron.Schedule
	if cf.Downloader.Enabled {
		var err error
		schedule, err = cron.ParseStandard(cf.Downloader.Schedule)
		if err != nil {
			return errors.New("configuration: " + err.Error())
		}
	}

	conf.Organizer = cf.Organizer
	conf.Logger = cf.Logs.logger
	conf.Watcher = WatcherConfig{
		Dir:        cf.Watcher.Dir,
		FsNotifier: cf.Watcher.fsNotifier,
	}
	conf.Downloader = DownloaderConfig{
		Enabled:         cf.Downloader.Enabled,
		LaunchAtStartup: cf.Downloader.LaunchAtStartup,
		Schedule:        schedule,
		Client:          cf.Downloader.downloader,
	}
	conf.DownloadManager = cf.DownloadManager
	conf.HTTPServer = cf.HTTPServer
	conf.Wishlist = polochon.WishlistConfig{
		Wishlisters:           cf.Wishlist.wishlisters,
		ShowDefaultQualities:  cf.Wishlist.ShowDefaultQualities,
		MovieDefaultQualities: cf.Wishlist.MovieDefaultQualities,
	}
	conf.Movie = polochon.MovieConfig{
		Detailers:  cf.Movie.detailers,
		Torrenters: cf.Movie.torrenters,
		Subtitlers: cf.Movie.subtitlers,
		Explorers:  cf.Movie.explorers,
		Searchers:  cf.Movie.searchers,
	}
	conf.Show = polochon.ShowConfig{
		Detailers:  cf.Show.detailers,
		Torrenters: cf.Show.torrenters,
		Subtitlers: cf.Show.subtitlers,
		Explorers:  cf.Show.explorers,
		Searchers:  cf.Show.searchers,
		Calendar:   cf.Show.calendar,
	}
	conf.File = polochon.FileConfig{
		ExcludeFileContaining:     cf.Video.ExcludeFileContaining,
		VideoExtensions:           cf.Video.VideoExtensions,
		AllowedExtensionsToDelete: cf.Video.AllowedExtensionsToDelete,
		Guessers:                  cf.Video.guessers,
	}
	conf.Library = LibraryConfig{}
	conf.Notifiers = cf.Video.notifiers
	conf.SubtitleLanguages = cf.Video.SubtitleLanguages

	// Check the default show qualities
	if err := checkQuality(conf.Wishlist.ShowDefaultQualities); err != nil {
		return err
	}

	if err := checkQuality(conf.Wishlist.MovieDefaultQualities); err != nil {
		return err
	}

	if err := evalSymlink(&conf.Library.MovieDir, cf.Movie.Dir); err != nil {
		return err
	}

	if err := evalSymlink(&conf.Library.ShowDir, cf.Show.Dir); err != nil {
		return err
	}

	return nil
}
