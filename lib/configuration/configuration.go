package configuration

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ConfigFileRoot represents polochon's config file
type ConfigFileRoot struct {
	Logs          ConfigFileLogs           `yaml:"logs"`
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
func (c *ConfigFileRoot) moduleParams(moduleName string) ([]byte, error) {
	for _, p := range c.ModulesParams {
		// Is the name of the module missing in the conf ?
		name, ok := p["name"]
		if !ok {
			return nil, fmt.Errorf("config: missing module name in configuration params: %+v", p)
		}

		// Not the right module name
		if moduleName != name {
			continue
		}

		// Encode the params using the yaml format so that each module can
		// decode it itself
		return yaml.Marshal(p)
	}

	// Nothing found
	return nil, nil
}

// ConfigFileVideo represents the configuration for the video in the configuration file
type ConfigFileVideo struct {
	GuesserName               string              `yaml:"guesser"`
	NotifierNames             []string            `yaml:"notifiers"`
	ExcludeFileContaining     []string            `yaml:"exclude_file_containing"`
	VideoExtentions           []string            `yaml:"allowed_file_extensions"`
	AllowedExtentionsToDelete []string            `yaml:"allowed_file_extensions_to_delete"`
	SubtitleLanguages         []polochon.Language `yaml:"subtitle_languages"`
}

// ConfigFileLogs represents the configuration for the logs of the app
type ConfigFileLogs struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// ConfigFileWatcher represents the configuration for the file watcher in the configuration file
type ConfigFileWatcher struct {
	Dir            string `yaml:"dir"`
	FsNotifierName string `yaml:"fsnotifier"`
}

// ConfigFileWishlist represents the configuration for the wishlist in the configuration file
type ConfigFileWishlist struct {
	WishlisterNames       []string           `yaml:"wishlisters"`
	ShowDefaultQualities  []polochon.Quality `yaml:"show_default_qualities"`
	MovieDefaultQualities []polochon.Quality `yaml:"movie_default_qualities"`
}

// ConfigFileMovie represents the configuration for movies in the configuration file
type ConfigFileMovie struct {
	Dir            string   `yaml:"dir"`
	TorrenterNames []string `yaml:"torrenters"`
	DetailerNames  []string `yaml:"detailers"`
	SubtitlerNames []string `yaml:"subtitlers"`
	SearcherNames  []string `yaml:"searchers"`
	ExplorerNames  []string `yaml:"explorers"`
}

// ConfigFileShow represents the configuration for file in the configuration file
type ConfigFileShow struct {
	Dir            string   `yaml:"dir"`
	TorrenterNames []string `yaml:"torrenters"`
	DetailerNames  []string `yaml:"detailers"`
	SubtitlerNames []string `yaml:"subtitlers"`
	SearcherNames  []string `yaml:"searchers"`
	ExplorerNames  []string `yaml:"explorers"`
	CalendarName   string   `yaml:"calendar"`
}

// ConfigFileDownloader represents the configuration for the downloader in the configuration file
type ConfigFileDownloader struct {
	Enabled        bool              `yaml:"enabled"`
	Timer          time.Duration     `yaml:"timer"`
	DownloaderName string            `yaml:"client"`
	Cleaner        ConfigFileCleaner `yaml:"cleaner"`
}

// ConfigFileCleaner represents the configuration for the downloader in the configuration file
type ConfigFileCleaner struct {
	Enabled bool          `yaml:"enabled"`
	Timer   time.Duration `yaml:"timer"`
	Ratio   float32       `yaml:"ratio"`
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
	Logger            *logrus.Logger
	Watcher           WatcherConfig
	Downloader        DownloaderConfig
	HTTPServer        HTTPServerConfig
	ModulesParams     []map[string]interface{}
	Wishlist          polochon.WishlistConfig
	Movie             polochon.MovieConfig
	Show              polochon.ShowConfig
	File              polochon.FileConfig
	Library           LibraryConfig
	Notifiers         []polochon.Notifier
	SubtitleLanguages []polochon.Language
}

// ModuleFetcher is an interface which allows to get torrenters and detailers ...
type ModuleFetcher interface {
	GetDetailers() []polochon.Detailer
	GetTorrenters() []polochon.Torrenter
	GetSearchers() []polochon.Searcher
	GetExplorers() []polochon.Explorer
	GetSubtitlers() []polochon.Subtitler
}

// LibraryConfig represents configuration for the library
type LibraryConfig struct {
	MovieDir string
	ShowDir  string
}

// WatcherConfig represents the configuration for the detailers
type WatcherConfig struct {
	Dir        string
	FsNotifier polochon.FsNotifier
}

// DownloaderConfig represents the configuration for the downloader
type DownloaderConfig struct {
	Enabled bool
	Timer   time.Duration
	Client  polochon.Downloader
	Cleaner CleanerConfig
}

// CleanerConfig represents the configuration for the cleaner in the configuration file
type CleanerConfig struct {
	Enabled bool
	Timer   time.Duration
	Ratio   float32
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

func loadConfig(cf *ConfigFileRoot) (*Config, error) {
	conf := &Config{}

	// Setup the logger
	logger, err := cf.loadLogger()
	if err != nil {
		return nil, err
	}
	conf.Logger = logger
	log := logrus.NewEntry(logger)

	conf.Watcher = WatcherConfig{
		Dir: cf.Watcher.Dir,
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

	notifiers, err := cf.initNotifiers()
	if err != nil {
		return nil, err
	}

	conf.Notifiers = notifiers

	wishlistConf, err := cf.initWishlist(log)
	if err != nil {
		return nil, err
	}
	conf.Wishlist = *wishlistConf

	err = conf.InitShow(cf)
	if err != nil {
		return nil, err
	}

	err = conf.InitMovie(cf)
	if err != nil {
		return nil, err
	}

	guesser, err := cf.initFile(log)
	if err != nil {
		return nil, err
	}

	conf.File = polochon.FileConfig{
		ExcludeFileContaining:     cf.Video.ExcludeFileContaining,
		VideoExtentions:           cf.Video.VideoExtentions,
		AllowedExtentionsToDelete: cf.Video.AllowedExtentionsToDelete,
		Guesser:                   guesser,
	}

	realShowsPath, err := filepath.EvalSymlinks(cf.Show.Dir)
	if err != nil {
		return nil, err
	}

	realMoviesPath, err := filepath.EvalSymlinks(cf.Movie.Dir)
	if err != nil {
		return nil, err
	}

	conf.Library = LibraryConfig{
		MovieDir: realMoviesPath,
		ShowDir:  realShowsPath,
	}

	conf.SubtitleLanguages = cf.Video.SubtitleLanguages

	return conf, nil
}

func (c *ConfigFileRoot) loadLogger() (*logrus.Logger, error) {
	// Create a new logger
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
	}

	// Get the log level
	logLevel, err := logrus.ParseLevel(c.Logs.Level)
	if err != nil {
		return nil, err
	}
	logger.Level = logLevel

	// Setup the output file
	var logOut io.Writer
	if c.Logs.File == "" {
		logOut = os.Stderr
	} else {
		var err error
		logOut, err = os.OpenFile(c.Logs.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			return nil, err
		}
	}
	logger.Out = logOut

	return logger, nil
}

func (c *ConfigFileRoot) loadWatcher(log *logrus.Entry) (polochon.FsNotifier, error) {
	if c.Watcher.FsNotifierName == "" {
		return nil, fmt.Errorf("config: missing watcher fsnotifier name")
	}

	// get params
	moduleParams, err := c.moduleParams(c.Watcher.FsNotifierName)
	if err != nil {
		return nil, err
	}

	// Configure
	fsNotifier, err := polochon.ConfigureFsNotifier(c.Watcher.FsNotifierName, moduleParams)
	if err != nil {
		return nil, err
	}

	return fsNotifier, nil
}

func (c *ConfigFileRoot) initFile(log *logrus.Entry) (polochon.Guesser, error) {
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
	guesser, err := polochon.ConfigureGuesser(c.Video.GuesserName, moduleParams)
	if err != nil {
		return nil, err
	}

	return guesser, nil
}

func (c *ConfigFileRoot) initWishlist(log *logrus.Entry) (*polochon.WishlistConfig, error) {
	wishlistConfig := &polochon.WishlistConfig{}

	// Configure the wishlisters
	for _, wishlisterName := range c.Wishlist.WishlisterNames {
		moduleParams, err := c.moduleParams(wishlisterName)
		if err != nil {
			return nil, err
		}

		wishlister, err := polochon.ConfigureWishlister(wishlisterName, moduleParams)
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
		Timer:   c.Downloader.Timer,
		Enabled: c.Downloader.Enabled,
	}

	if c.Downloader.DownloaderName != "" && c.Downloader.Enabled {
		moduleParams, err := c.moduleParams(c.Downloader.DownloaderName)
		if err != nil {
			return nil, err
		}

		downloader, err := polochon.ConfigureDownloader(c.Downloader.DownloaderName, moduleParams)
		if err != nil {
			return nil, err
		}
		downloaderConf.Client = downloader
	}

	cleanerConf, err := c.initCleaner(log)
	if err != nil {
		return nil, err
	}
	downloaderConf.Cleaner = *cleanerConf

	return downloaderConf, nil
}

func (c *ConfigFileRoot) initCleaner(log *logrus.Entry) (*CleanerConfig, error) {
	cleanerConf := &CleanerConfig{
		Timer:   c.Downloader.Cleaner.Timer,
		Enabled: c.Downloader.Cleaner.Enabled,
		Ratio:   c.Downloader.Cleaner.Ratio,
	}

	return cleanerConf, nil
}

func (c *ConfigFileRoot) initNotifiers() ([]polochon.Notifier, error) {
	notifiers := []polochon.Notifier{}

	for _, notifierName := range c.Video.NotifierNames {
		moduleParams, err := c.moduleParams(notifierName)
		if err != nil {
			return nil, err
		}

		notifier, err := polochon.ConfigureNotifier(notifierName, moduleParams)
		if err != nil {
			return nil, err
		}
		notifiers = append(notifiers, notifier)
	}

	return notifiers, nil
}

// InitShow inits the show's config
func (c *Config) InitShow(cf *ConfigFileRoot) error {
	// Get show detailer
	if len(cf.Show.DetailerNames) == 0 {
		return fmt.Errorf("config: missing show detailer names")
	}
	showConf := polochon.ShowConfig{}
	for _, detailerName := range cf.Show.DetailerNames {
		moduleParams, err := cf.moduleParams(detailerName)
		if err != nil {
			return err
		}

		detailer, err := polochon.ConfigureDetailer(detailerName, moduleParams)
		if err != nil {
			return err
		}

		showConf.Detailers = append(showConf.Detailers, detailer)
	}

	// Get show torrenter
	if len(cf.Show.TorrenterNames) == 0 {
		return fmt.Errorf("config: missing movie torrenter names")
	}

	for _, torrenterName := range cf.Show.TorrenterNames {
		moduleParams, err := cf.moduleParams(torrenterName)
		if err != nil {
			return err
		}

		torrenter, err := polochon.ConfigureTorrenter(torrenterName, moduleParams)
		if err != nil {
			return err
		}
		showConf.Torrenters = append(showConf.Torrenters, torrenter)
	}

	for _, subtitlerName := range cf.Show.SubtitlerNames {
		moduleParams, err := cf.moduleParams(subtitlerName)
		if err != nil {
			return err
		}

		subtitler, err := polochon.ConfigureSubtitler(subtitlerName, moduleParams)
		if err != nil {
			return err
		}
		showConf.Subtitlers = append(showConf.Subtitlers, subtitler)
	}

	for _, explorerName := range cf.Show.ExplorerNames {
		moduleParams, err := cf.moduleParams(explorerName)
		if err != nil {
			return err
		}

		explorer, err := polochon.ConfigureExplorer(explorerName, moduleParams)
		if err != nil {
			return err
		}
		showConf.Explorers = append(showConf.Explorers, explorer)
	}

	for _, searcherName := range cf.Show.SearcherNames {
		moduleParams, err := cf.moduleParams(searcherName)
		if err != nil {
			return err
		}

		searcher, err := polochon.ConfigureSearcher(searcherName, moduleParams)
		if err != nil {
			return err
		}
		showConf.Searchers = append(showConf.Searchers, searcher)
	}

	// Init the show calendar fetcher
	if cf.Show.CalendarName != "" {
		moduleParams, err := cf.moduleParams(cf.Show.CalendarName)
		if err != nil {
			return err
		}

		// Configure
		calendar, err := polochon.ConfigureCalendar(cf.Show.CalendarName, moduleParams)
		if err != nil {
			return err
		}

		showConf.Calendar = calendar
	}

	c.Show = showConf

	return nil
}

// InitMovie inits the movie's config
func (c *Config) InitMovie(cf *ConfigFileRoot) error {
	// Get movie detailer
	if len(cf.Movie.DetailerNames) == 0 {
		return fmt.Errorf("config: missing movie detailer names")
	}

	movieConf := polochon.MovieConfig{}

	for _, detailerName := range cf.Movie.DetailerNames {
		moduleParams, err := cf.moduleParams(detailerName)
		if err != nil {
			return err
		}

		detailer, err := polochon.ConfigureDetailer(detailerName, moduleParams)
		if err != nil {
			return err
		}
		movieConf.Detailers = append(movieConf.Detailers, detailer)
	}

	// Get movie torrenter
	if len(cf.Movie.TorrenterNames) == 0 {
		return fmt.Errorf("config: missing movie torrenter names")
	}

	for _, torrenterName := range cf.Movie.TorrenterNames {
		moduleParams, err := cf.moduleParams(torrenterName)
		if err != nil {
			return err
		}

		torrenter, err := polochon.ConfigureTorrenter(torrenterName, moduleParams)
		if err != nil {
			return err
		}
		movieConf.Torrenters = append(movieConf.Torrenters, torrenter)
	}

	for _, subtitlerName := range cf.Movie.SubtitlerNames {
		moduleParams, err := cf.moduleParams(subtitlerName)
		if err != nil {
			return err
		}

		subtitler, err := polochon.ConfigureSubtitler(subtitlerName, moduleParams)
		if err != nil {
			return err
		}

		movieConf.Subtitlers = append(movieConf.Subtitlers, subtitler)
	}

	for _, explorerName := range cf.Movie.ExplorerNames {
		moduleParams, err := cf.moduleParams(explorerName)
		if err != nil {
			return err
		}

		explorer, err := polochon.ConfigureExplorer(explorerName, moduleParams)
		if err != nil {
			return err
		}
		movieConf.Explorers = append(movieConf.Explorers, explorer)
	}

	for _, searcherName := range cf.Movie.SearcherNames {
		moduleParams, err := cf.moduleParams(searcherName)
		if err != nil {
			return err
		}

		searcher, err := polochon.ConfigureSearcher(searcherName, moduleParams)
		if err != nil {
			return err
		}
		movieConf.Searchers = append(movieConf.Searchers, searcher)
	}

	c.Movie = movieConf

	return nil
}

// ModuleStatus represent the status of a module
type ModuleStatus struct {
	Name   string                `json:"name"`
	Status polochon.ModuleStatus `json:"status"`
	Error  string                `json:"error"`
}

// ModulesStatuses represent the status of all the modules
type ModulesStatuses map[string]map[string][]ModuleStatus

func errorMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ModulesStatus gives the status of the modules configured
func (c *Config) ModulesStatus() ModulesStatuses {
	type module struct {
		moduleType string
		config     ModuleFetcher
	}

	result := ModulesStatuses{}
	for _, module := range []module{
		{
			"movie",
			&c.Movie,
		},
		{
			"show",
			&c.Show,
		},
	} {
		result[module.moduleType] = map[string][]ModuleStatus{}
		for _, detailer := range module.config.GetDetailers() {
			status, err := detailer.Status()
			result[module.moduleType]["detailer"] = append(result[module.moduleType]["detailer"], ModuleStatus{
				Name:   detailer.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
		for _, subtitler := range module.config.GetSubtitlers() {
			status, err := subtitler.Status()
			result[module.moduleType]["subtitler"] = append(result[module.moduleType]["subtitler"], ModuleStatus{
				Name:   subtitler.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
		for _, torrenter := range module.config.GetTorrenters() {
			status, err := torrenter.Status()
			result[module.moduleType]["torrenter"] = append(result[module.moduleType]["torrenter"], ModuleStatus{
				Name:   torrenter.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
		for _, explorer := range module.config.GetExplorers() {
			status, err := explorer.Status()
			result[module.moduleType]["explorer"] = append(result[module.moduleType]["explorer"], ModuleStatus{
				Name:   explorer.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
		for _, searcher := range module.config.GetSearchers() {
			status, err := searcher.Status()
			result[module.moduleType]["searcher"] = append(result[module.moduleType]["searcher"], ModuleStatus{
				Name:   searcher.Name(),
				Status: status,
				Error:  errorMsg(err),
			})
		}
	}

	return result
}

// LoadConfigFile reads a file from a path and returns a config
func LoadConfigFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	cf, err := readConfig(file)
	if err != nil {
		return nil, err
	}
	return loadConfig(cf)
}
