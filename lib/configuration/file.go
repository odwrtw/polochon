package configuration

import (
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

type configFile struct {
	modulesParams *ModulesParams

	Logs Logger `yaml:"logs"`

	Watcher struct {
		ModuleLoader `yaml:",inline"`
		Dir          string `yaml:"dir"`
	} `yaml:"watcher"`

	Downloader struct {
		ModuleLoader `yaml:",inline"`
		Enabled      bool          `yaml:"enabled"`
		Timer        time.Duration `yaml:"timer"`
		Cleaner      CleanerConfig `yaml:"cleaner"`
	} `yaml:"downloader"`

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

	conf.Logger = cf.Logs.logger
	conf.Watcher = WatcherConfig{
		Dir:        cf.Watcher.Dir,
		FsNotifier: cf.Watcher.fsNotifier,
	}
	conf.Downloader = DownloaderConfig{
		Enabled: cf.Downloader.Enabled,
		Timer:   cf.Downloader.Timer,
		Client:  cf.Downloader.downloader,
		Cleaner: cf.Downloader.Cleaner,
	}
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
		Guesser:                   cf.Video.guesser,
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
