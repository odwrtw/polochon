package polochon

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ConfigFileRoot represents polochon's config file
type ConfigFileRoot struct {
	Watcher       ConfigFileWatcher        `yaml:"watcher"`
	Downloader    ConfigFileDownloader     `yaml:"downloader"`
	HTTPServer    ConfigFileHTTPServer     `yaml:"http_server"`
	ModulesParams []map[string]interface{} `yaml:"modules_params"`
	Video         ConfigFileVideo          `yaml:"video"`
	Show          ConfigFileShow           `yaml:"show"`
	Movie         ConfigFileMovie          `yaml:"movie"`
	Wishlist      ConfigFileWishlist       `yaml:"wishlist"`
}

// moduleParams returns the modules params set in the configuration.
func (c *ConfigFileRoot) moduleParams(moduleName string) (map[string]interface{}, error) {
	for _, p := range c.ModulesParams {
		// Is the name of the module missing in the conf ?
		name, ok := p["name"]
		if !ok {
			return map[string]interface{}{}, fmt.Errorf("config: missing module name in configuration params: %+v", p)
		}

		// Found the right module config
		if moduleName == name {
			return p, nil
		}
	}

	// Nothing found, return the default values
	return map[string]interface{}{}, nil
}

// ConfigFileVideo represents the configuration for the video in the configuration file
type ConfigFileVideo struct {
	GuesserName               string   `yaml:"guesser"`
	NotifierNames             []string `yaml:"notifiers"`
	ExcludeFileContaining     []string `yaml:"exclude_file_containing"`
	VideoExtentions           []string `yaml:"allowed_file_extensions"`
	AllowedExtentionsToDelete []string `yaml:"allowed_file_extensions_to_delete"`
}

// ConfigFileWatcher represents the configuration for the file watcher in the configuration file
type ConfigFileWatcher struct {
	Timer          time.Duration `yaml:"timer"`
	Dir            string        `yaml:"dir"`
	FsNotifierName string        `yaml:"fsnotifier"`
}

// ConfigFileWishlist represents the configuration for the wishlist in the configuration file
type ConfigFileWishlist struct {
	WishlisterNames       []string  `yaml:"wishlisters"`
	ShowDefaultQualities  []Quality `yaml:"show_default_qualities"`
	MovieDefaultQualities []Quality `yaml:"movie_default_qualities"`
}

// ConfigFileMovie represents the configuration for movies in the configuration file
type ConfigFileMovie struct {
	Dir            string   `yaml:"dir"`
	TorrenterNames []string `yaml:"torrenters"`
	DetailerNames  []string `yaml:"detailers"`
	SubtitlerNames []string `yaml:"subtitlers"`
}

// ConfigFileShow represents the configuration for file in the configuration file
type ConfigFileShow struct {
	Dir            string   `yaml:"dir"`
	TorrenterNames []string `yaml:"torrenters"`
	DetailerNames  []string `yaml:"detailers"`
	SubtitlerNames []string `yaml:"subtitlers"`
}

// ConfigFileDownloader represents the configuration for the downloader in the configuration file
type ConfigFileDownloader struct {
	DownloadDir    string `yaml:"download_dir"`
	DownloaderName string `yaml:"client"`
}

// ConfigFileHTTPServer represents the configuration for the HTTP Server in the configuration file
type ConfigFileHTTPServer struct {
	Enable            bool   `yaml:"enable"`
	Port              int    `yaml:"port"`
	Host              string `yaml:"host"`
	ServeFiles        bool   `yaml:"serve_files"`
	BasicAuth         bool   `yaml:"basic_auth"`
	BasicAuthUser     string `yaml:"basic_auth_user"`
	BasicAuthPassword string `yaml:"basic_auth_password"`
}

// Config represents the configuration for polochon
type Config struct {
	Watcher       WatcherConfig
	Downloader    DownloaderConfig
	HTTPServer    HTTPServerConfig
	ModulesParams []map[string]interface{}
	Video         VideoConfig
	Wishlist      WishlistConfig
	File          FileConfig
}

// WatcherConfig represents the configuration for the detailers
type WatcherConfig struct {
	Timer      time.Duration
	Dir        string
	FsNotifier FsNotifier
}

// DownloaderConfig represents the configuration for the downloader
type DownloaderConfig struct {
	DownloadDir string
	Client      Downloader
}

// HTTPServerConfig represents the configuration for the HTTP Server
type HTTPServerConfig struct {
	Enable            bool
	Port              int
	Host              string
	ServeFiles        bool
	BasicAuth         bool
	BasicAuthUser     string
	BasicAuthPassword string
}

// VideoConfig represents the configuration for video object
type VideoConfig struct {
	Notifiers []Notifier
	Show      ShowConfig
	Movie     MovieConfig
}

// WishlistConfig represents the wishlist configurations
type WishlistConfig struct {
	Wishlisters           []Wishlister
	ShowDefaultQualities  []Quality `yaml:"show_default_qualities"`
	MovieDefaultQualities []Quality `yaml:"movie_default_qualities"`
}

// ShowConfig represents the configuration for a show and its show episodes
type ShowConfig struct {
	Dir        string
	Detailers  []Detailer
	Notifiers  []Notifier
	Subtitlers []Subtitler
	Torrenters []Torrenter
}

// MovieConfig represents the configuration for a movie
type MovieConfig struct {
	Dir        string
	Torrenters []Torrenter
	Detailers  []Detailer
	Subtitlers []Subtitler
	Notifiers  []Notifier
}

// FileConfig represents the configuration for a file
type FileConfig struct {
	ExcludeFileContaining     []string
	VideoExtentions           []string
	AllowedExtentionsToDelete []string
	Guesser                   Guesser
}

// readConfig helps read the config
func readConfig(r io.Reader) (*ConfigFileRoot, error) {
	cf := &ConfigFileRoot{}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, cf)
	if err != nil {
		return nil, err
	}

	return cf, nil
}

func loadConfig(cf *ConfigFileRoot, log *logrus.Entry) (*Config, error) {
	conf := &Config{}

	conf.Watcher = WatcherConfig{
		Timer: cf.Watcher.Timer,
		Dir:   cf.Watcher.Dir,
	}

	fsNotifier, err := cf.loadWatcher(log)
	if err != nil {
		return nil, err
	}
	conf.Watcher.FsNotifier = fsNotifier

	conf.HTTPServer = HTTPServerConfig{
		Enable:            cf.HTTPServer.Enable,
		Port:              cf.HTTPServer.Port,
		Host:              cf.HTTPServer.Host,
		ServeFiles:        cf.HTTPServer.ServeFiles,
		BasicAuth:         cf.HTTPServer.BasicAuth,
		BasicAuthUser:     cf.HTTPServer.BasicAuthUser,
		BasicAuthPassword: cf.HTTPServer.BasicAuthPassword,
	}

	conf.ModulesParams = cf.ModulesParams

	downloaderConf, err := cf.initDownloader(log)
	if err != nil {
		return nil, err
	}
	conf.Downloader = *downloaderConf

	videoConf, err := cf.initVideo(log)
	if err != nil {
		return nil, err
	}
	conf.Video = *videoConf

	wishlistConf, err := cf.initWishlist(log)
	if err != nil {
		return nil, err
	}
	conf.Wishlist = *wishlistConf

	showConf, err := cf.initShow(log)
	if err != nil {
		return nil, err
	}

	realShowsPath, err := filepath.EvalSymlinks(cf.Show.Dir)
	if err != nil {
		return nil, err
	}

	showConf.Dir = realShowsPath

	showConf.Notifiers = conf.Video.Notifiers
	conf.Video.Show = *showConf

	movieConf, err := cf.initMovie(log)
	if err != nil {
		return nil, err
	}

	realMoviesPath, err := filepath.EvalSymlinks(cf.Movie.Dir)
	if err != nil {
		return nil, err
	}

	movieConf.Dir = realMoviesPath

	movieConf.Notifiers = conf.Video.Notifiers
	conf.Video.Movie = *movieConf

	guesser, err := cf.initFile(log)
	if err != nil {
		return nil, err
	}

	conf.File = FileConfig{
		ExcludeFileContaining:     cf.Video.ExcludeFileContaining,
		VideoExtentions:           cf.Video.VideoExtentions,
		AllowedExtentionsToDelete: cf.Video.AllowedExtentionsToDelete,
		Guesser:                   guesser,
	}

	return conf, nil
}

func (c *ConfigFileRoot) loadWatcher(log *logrus.Entry) (FsNotifier, error) {
	if c.Watcher.FsNotifierName == "" {
		return nil, fmt.Errorf("config: missing watcher fsnotifier name")
	}

	// get params
	moduleParams, err := c.moduleParams(c.Watcher.FsNotifierName)
	if err != nil {
		return nil, err
	}

	// Configure
	fsNotifier, err := ConfigureFsNotifier(c.Watcher.FsNotifierName, moduleParams, log)
	if err != nil {
		return nil, err
	}

	return fsNotifier, nil
}
func (c *ConfigFileRoot) initFile(log *logrus.Entry) (Guesser, error) {
	// Get video guesser
	if c.Video.GuesserName == "" {
		return nil, fmt.Errorf("config: missing video guesser name")
	}

	// get params
	moduleParams, err := c.moduleParams(c.Video.GuesserName)
	if err != nil {
		return nil, err
	}

	// Configure
	guesser, err := ConfigureGuesser(c.Video.GuesserName, moduleParams, log)
	if err != nil {
		return nil, err
	}

	return guesser, nil
}

func (c *ConfigFileRoot) initWishlist(log *logrus.Entry) (*WishlistConfig, error) {
	wishlistConfig := &WishlistConfig{}

	// Configure the wishlisters
	for _, wishlisterName := range c.Wishlist.WishlisterNames {
		moduleParams, err := c.moduleParams(wishlisterName)
		if err != nil {
			return nil, err
		}

		wishlister, err := ConfigureWishlister(wishlisterName, moduleParams, log)
		if err != nil {
			return nil, err
		}
		wishlistConfig.Wishlisters = append(wishlistConfig.Wishlisters, wishlister)
	}

	// Check the default show qualities
	for _, q := range c.Wishlist.ShowDefaultQualities {
		if !q.IsAllowed() {
			return nil, fmt.Errorf("wishlist config: invalid show quality: %q", q)
		}
		wishlistConfig.ShowDefaultQualities = append(wishlistConfig.ShowDefaultQualities, q)
	}

	// Check the default movie qualities
	for _, q := range c.Wishlist.MovieDefaultQualities {
		if !q.IsAllowed() {
			return nil, fmt.Errorf("wishlist config: invalid movie quality: %q", q)
		}
		wishlistConfig.MovieDefaultQualities = append(wishlistConfig.MovieDefaultQualities, q)
	}

	return wishlistConfig, nil
}

func (c *ConfigFileRoot) initDownloader(log *logrus.Entry) (*DownloaderConfig, error) {
	downloaderConf := &DownloaderConfig{
		DownloadDir: c.Downloader.DownloadDir,
	}

	if c.Downloader.DownloaderName != "" {
		moduleParams, err := c.moduleParams(c.Downloader.DownloaderName)
		if err != nil {
			return nil, err
		}

		downloader, err := ConfigureDownloader(c.Downloader.DownloaderName, moduleParams, log)
		if err != nil {
			return nil, err
		}
		downloaderConf.Client = downloader
	}

	return downloaderConf, nil
}

func (c *ConfigFileRoot) initVideo(log *logrus.Entry) (*VideoConfig, error) {

	videoConf := &VideoConfig{}

	for _, notifierName := range c.Video.NotifierNames {
		moduleParams, err := c.moduleParams(notifierName)
		if err != nil {
			return nil, err
		}

		notifier, err := ConfigureNotifier(notifierName, moduleParams, log)
		if err != nil {
			return nil, err
		}
		videoConf.Notifiers = append(videoConf.Notifiers, notifier)
	}

	return videoConf, nil
}

func (c *ConfigFileRoot) initShow(log *logrus.Entry) (*ShowConfig, error) {
	// Get show detailer
	if len(c.Show.DetailerNames) == 0 {
		return nil, fmt.Errorf("config: missing show detailer names")
	}
	showConf := &ShowConfig{}
	for _, detailerName := range c.Show.DetailerNames {
		moduleParams, err := c.moduleParams(detailerName)
		if err != nil {
			return nil, err
		}

		detailer, err := ConfigureDetailer(detailerName, moduleParams, log)
		if err != nil {
			return nil, err
		}

		showConf.Detailers = append(showConf.Detailers, detailer)
	}

	// Get show torrenter
	if len(c.Show.TorrenterNames) == 0 {
		return nil, fmt.Errorf("config: missing movie torrenter names")
	}

	for _, torrenterName := range c.Show.TorrenterNames {
		moduleParams, err := c.moduleParams(torrenterName)
		if err != nil {
			return nil, err
		}

		torrenter, err := ConfigureTorrenter(torrenterName, moduleParams, log)
		if err != nil {
			return nil, err
		}
		showConf.Torrenters = append(showConf.Torrenters, torrenter)
	}

	for _, subtitlerName := range c.Show.SubtitlerNames {
		moduleParams, err := c.moduleParams(subtitlerName)
		if err != nil {
			return nil, err
		}

		subtitler, err := ConfigureSubtitler(subtitlerName, moduleParams, log)
		if err != nil {
			return nil, err
		}
		showConf.Subtitlers = append(showConf.Subtitlers, subtitler)
	}

	return showConf, nil
}

func (c *ConfigFileRoot) initMovie(log *logrus.Entry) (*MovieConfig, error) {
	// Get movie detailer
	if len(c.Movie.DetailerNames) == 0 {
		return nil, fmt.Errorf("config: missing movie detailer names")
	}

	movieConf := &MovieConfig{}

	for _, detailerName := range c.Movie.DetailerNames {
		moduleParams, err := c.moduleParams(detailerName)
		if err != nil {
			return nil, err
		}

		detailer, err := ConfigureDetailer(detailerName, moduleParams, log)
		if err != nil {
			return nil, err
		}
		movieConf.Detailers = append(movieConf.Detailers, detailer)
	}

	// Get movie torrenter
	if len(c.Movie.TorrenterNames) == 0 {
		return nil, fmt.Errorf("config: missing movie torrenter names")
	}

	for _, torrenterName := range c.Movie.TorrenterNames {
		moduleParams, err := c.moduleParams(torrenterName)
		if err != nil {
			return nil, err
		}

		torrenter, err := ConfigureTorrenter(torrenterName, moduleParams, log)
		if err != nil {
			return nil, err
		}
		movieConf.Torrenters = append(movieConf.Torrenters, torrenter)
	}

	for _, subtitlerName := range c.Movie.SubtitlerNames {
		moduleParams, err := c.moduleParams(subtitlerName)
		if err != nil {
			return nil, err
		}

		subtitler, err := ConfigureSubtitler(subtitlerName, moduleParams, log)
		if err != nil {
			return nil, err
		}

		movieConf.Subtitlers = append(movieConf.Subtitlers, subtitler)
	}

	return movieConf, nil
}

// LoadConfigFile reads a file from a path and returns a config
func LoadConfigFile(path string, log *logrus.Entry) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	cf, err := readConfig(file)
	if err != nil {
		return nil, err
	}
	return loadConfig(cf, log)
}
