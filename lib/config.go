package polochon

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ConfigFileMain represents polochon's config file
type ConfigFileRoot struct {
	Watcher       ConfigFileWatcher    `yaml:"watcher"`
	Downloader    ConfigFileDownloader `yaml:"downloader"`
	HTTPServer    ConfigFileHTTPServer `yaml:"http_server"`
	ModulesParams []map[string]string  `yaml:"modules_params"`
	Video         ConfigFileVideo      `yaml:"video"`
	Show          ConfigFileShow       `yaml:"show"`
	Movie         ConfigFileMovie      `yaml:"movie"`
}

// moduleParams returuns the modules params set in the configuration.
func (c *ConfigFileRoot) moduleParams(moduleName string) (map[string]string, error) {
	for _, p := range c.ModulesParams {
		// Is the name of the module missing in the conf ?
		name, ok := p["name"]
		if !ok {
			return map[string]string{}, fmt.Errorf("config: missing module name in configuration params: %+v", p)
		}

		// Found the right module config
		if moduleName == name {
			return p, nil
		}
	}

	// Nothing found, return the default values
	return map[string]string{}, nil
}

type ConfigFileVideo struct {
	GuesserName               string   `yaml:"guesser"`
	NotifierNames             []string `yaml:"notifiers"`
	ExcludeFileContaining     []string `yaml:"exclude_file_containing"`
	VideoExtentions           []string `yaml:"allowed_file_extensions"`
	AllowedExtentionsToDelete []string `yaml:"allowed_file_extensions_to_delete"`
}

type ConfigFileWatcher struct {
	Timer          time.Duration `yaml:"timer"`
	Dir            string        `yaml:"dir"`
	FsNotifierName string        `yaml:"fsnotifier"`
}

type ConfigFileMovie struct {
	Dir            string   `yaml:"dir"`
	TorrenterNames []string `yaml:"torrenters"`
	DetailerNames  []string `yaml:"detailers"`
}

type ConfigFileShow struct {
	Dir            string   `yaml:"dir"`
	TorrenterNames []string `yaml:"torrenters"`
	DetailerNames  []string `yaml:"detailers"`
	SubtitlerNames []string `yaml:"subtitilers"`
}

// DownloaderConfig represents the configuration for the downloader
type ConfigFileDownloader struct {
	DownloadDir string `yaml:"download_dir"`
}

// HTTPServerConfig represents the configuration for the HTTP Server
type ConfigFileHTTPServer struct {
	Enable         bool   `yaml:"enable"`
	Port           int    `yaml:"port"`
	Host           string `yaml:"host"`
	ServeFiles     bool   `yaml:"serve_files"`
	ServeFilesUser string `yaml:"serve_files_user"`
	ServeFilesPwd  string `yaml:"serve_files_pwd"`
}

type Config struct {
	Watcher       WatcherConfig
	Downloader    DownloaderConfig
	HTTPServer    HTTPServerConfig
	ModulesParams []map[string]string
	Video         VideoConfig
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
}

// HTTPServerConfig represents the configuration for the HTTP Server
type HTTPServerConfig struct {
	Enable         bool
	Port           int
	Host           string
	ServeFiles     bool
	ServeFilesUser string
	ServeFilesPwd  string
}

// VideoConfig represents the configuration for video object
type VideoConfig struct {
	Notifiers []Notifier
	Show      ShowConfig
	Movie     MovieConfig
}

// ShowConfig represents the configuration for a show and its show episodes
type ShowConfig struct {
	Dir         string
	Detailers   []Detailer
	Notifiers   []Notifier
	Subtitilers []Subtitiler
	Torrenters  []Torrenter
}

// MovieConfig represents the configuration for a movie
type MovieConfig struct {
	Dir        string
	Torrenters []Torrenter
	Detailers  []Detailer
	Notifiers  []Notifier
}

type FileConfig struct {
	ExcludeFileContaining     []string
	VideoExtentions           []string
	AllowedExtentionsToDelete []string
	Guesser                   Guesser
}

// readConfig helps read the config
func readConfig(r io.Reader, log *logrus.Entry) (*Config, error) {
	cf := &ConfigFileRoot{}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, cf)
	if err != nil {
		return nil, err
	}

	conf := &Config{}

	conf.Watcher = WatcherConfig{
		Timer: cf.Watcher.Timer,
		Dir:   cf.Watcher.Dir,
	}

	fsNotifier, err := cf.loadWatcher(log)
	if err != nil {
		return nil, err
	}
	conf.Watcher.FsNotifier = *fsNotifier

	conf.Downloader = DownloaderConfig{
		DownloadDir: cf.Downloader.DownloadDir,
	}

	conf.HTTPServer = HTTPServerConfig{
		Enable:         cf.HTTPServer.Enable,
		Port:           cf.HTTPServer.Port,
		Host:           cf.HTTPServer.Host,
		ServeFiles:     cf.HTTPServer.ServeFiles,
		ServeFilesUser: cf.HTTPServer.ServeFilesUser,
		ServeFilesPwd:  cf.HTTPServer.ServeFilesPwd,
	}

	conf.ModulesParams = cf.ModulesParams

	videoConf, err := cf.initVideo(log)
	if err != nil {
		return nil, err
	}
	conf.Video = *videoConf

	showConf, err := cf.initShow(log)
	if err != nil {
		return nil, err
	}
	showConf.Dir = cf.Show.Dir
	conf.Video.Show = *showConf

	movieConf, err := cf.initMovie(log)
	if err != nil {
		return nil, err
	}
	movieConf.Dir = cf.Movie.Dir
	conf.Video.Movie = *movieConf

	guesser, err := cf.initFile(log)
	if err != nil {
		return nil, err
	}

	conf.File = FileConfig{
		ExcludeFileContaining:     cf.Video.ExcludeFileContaining,
		VideoExtentions:           cf.Video.VideoExtentions,
		AllowedExtentionsToDelete: cf.Video.AllowedExtentionsToDelete,
		Guesser:                   *guesser,
	}

	return conf, nil
}

func (c *ConfigFileRoot) loadWatcher(log *logrus.Entry) (*FsNotifier, error) {
	if c.Watcher.FsNotifierName == "" {
		return nil, fmt.Errorf("config: missing watcher fsnotifier name")
	}

	// get params
	moduleParams, err := c.moduleParams(c.Watcher.FsNotifierName)
	if err != nil {
		return nil, err
	}

	// Configure
	if err := ConfiguredModules.ConfigureFsNotifier(c.Watcher.FsNotifierName, moduleParams, log); err != nil {
		return nil, err
	}

	n, ok := ConfiguredModules.FsNotifiers[c.Watcher.FsNotifierName]
	if !ok {
		return nil, ErrModuleNotFound
	}

	return &n, nil
}
func (c *ConfigFileRoot) initFile(log *logrus.Entry) (*Guesser, error) {
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
	if err := ConfiguredModules.ConfigureGuesser(c.Video.GuesserName, moduleParams, log); err != nil {
		return nil, err
	}

	g, ok := ConfiguredModules.Guessers[c.Video.GuesserName]
	if !ok {
		return nil, ErrModuleNotFound
	}
	return &g, nil
}

func (c *ConfigFileRoot) initVideo(log *logrus.Entry) (*VideoConfig, error) {

	videoConf := &VideoConfig{}

	for _, notifierName := range c.Video.NotifierNames {
		moduleParams, err := c.moduleParams(notifierName)
		if err != nil {
			return nil, err
		}

		if err := ConfiguredModules.ConfigureNotifier(notifierName, moduleParams, log); err != nil {
			return nil, err
		}

		n, ok := ConfiguredModules.Notifiers[notifierName]
		if !ok {
			return nil, ErrModuleNotFound
		}
		videoConf.Notifiers = append(videoConf.Notifiers, n)
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

		if err := ConfiguredModules.ConfigureDetailer(detailerName, moduleParams, log); err != nil {
			return nil, err
		}

		d, ok := ConfiguredModules.Detailers[detailerName]
		if !ok {
			return nil, ErrModuleNotFound
		}
		showConf.Detailers = append(showConf.Detailers, d)
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

		if err := ConfiguredModules.ConfigureTorrenter(torrenterName, moduleParams, log); err != nil {
			return nil, err
		}

		t, ok := ConfiguredModules.Torrenters[torrenterName]
		if !ok {
			return nil, ErrModuleNotFound
		}

		showConf.Torrenters = append(showConf.Torrenters, t)
	}

	for _, subtitlerName := range c.Show.SubtitlerNames {
		moduleParams, err := c.moduleParams(subtitlerName)
		if err != nil {
			return nil, err
		}

		if err := ConfiguredModules.ConfigureSubtitler(subtitlerName, moduleParams, log); err != nil {
			return nil, err
		}

		s, ok := ConfiguredModules.Subtitilers[subtitlerName]
		if !ok {
			return nil, ErrModuleNotFound
		}

		showConf.Subtitilers = append(showConf.Subtitilers, s)
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

		if err := ConfiguredModules.ConfigureDetailer(detailerName, moduleParams, log); err != nil {
			return nil, err
		}

		d, ok := ConfiguredModules.Detailers[detailerName]
		if !ok {
			return nil, ErrModuleNotFound
		}

		movieConf.Detailers = append(movieConf.Detailers, d)
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

		if err := ConfiguredModules.ConfigureTorrenter(torrenterName, moduleParams, log); err != nil {
			return nil, err
		}

		t, ok := ConfiguredModules.Torrenters[torrenterName]
		if !ok {
			return nil, ErrModuleNotFound
		}

		movieConf.Torrenters = append(movieConf.Torrenters, t)
	}

	return movieConf, nil
}

// writeConfig helps writes the config
// func (c *Config) write(w io.Writer) error {
// 	b, err := yaml.Marshal(c)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = w.Write(b)
// 	if err != nil {
// 		return err
// 	}

// 	return nil

// }

// ReadConfigFile reads a file from a path and returns a config
func LoadConfigFile(path string, log *logrus.Entry) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return readConfig(file, log)
}

// WriteConfigFile writes the config into file
// func (c *Config) WriteConfigFile(path string) error {
// 	file, err := os.Create(path)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	return c.write(file)
// }
