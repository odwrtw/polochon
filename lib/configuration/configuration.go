package configuration

import (
	"io"
	"os"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Config represents the configuration for polochon
type Config struct {
	Logger            *logrus.Logger
	Watcher           WatcherConfig
	Downloader        DownloaderConfig
	DownloadManager   DownloadManagerConfig
	HTTPServer        HTTPServer
	Wishlist          polochon.WishlistConfig
	Movie             polochon.MovieConfig
	Show              polochon.ShowConfig
	File              polochon.FileConfig
	Library           LibraryConfig
	Notifiers         []polochon.Notifier
	SubtitleLanguages []polochon.Language
}

// UnmarshalYAML implements the Unmarshaler interface
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Read the module params to be used later
	params := struct {
		ModulesParams *ModulesParams `yaml:"modules_params"`
	}{}

	if err := unmarshal(&params); err != nil {
		return err
	}

	// Read the rest of the file and use the module params to initiate the modules
	cf := &configFile{modulesParams: params.ModulesParams}
	if err := unmarshal(cf); err != nil {
		return err
	}

	// Load the configuration
	return loadConfig(cf, c)
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
	Enabled         bool
	LaunchAtStartup bool
	Schedule        cron.Schedule
	Client          polochon.Downloader
}

// DownloadManagerConfig represents the configuration for the download manager
type DownloadManagerConfig struct {
	Enabled bool          `yaml:"enabled"`
	Dir     string        `yaml:"dir"`
	Timer   time.Duration `yaml:"timer"`
	Ratio   float32       `yaml:"ratio"`
}

// HTTPServer represents the configuration for the HTTP Server
type HTTPServer struct {
	// TODO: enable => enabled
	Enable            bool     `yaml:"enable"`
	Port              int      `yaml:"port"`
	Host              string   `yaml:"host"`
	ServeFiles        bool     `yaml:"serve_files"`
	BasicAuth         bool     `yaml:"basic_auth"`
	BasicAuthUser     string   `yaml:"basic_auth_user"`
	BasicAuthPassword string   `yaml:"basic_auth_password"`
	LogExcludePaths   []string `yaml:"log_exclude_paths"`
}

// LoadConfig loads the configuration from a reader
func LoadConfig(r io.Reader) (*Config, error) {
	config := &Config{}
	return config, yaml.NewDecoder(r).Decode(config)
}

// LoadConfigFile reads a file from a path and returns a config
func LoadConfigFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return LoadConfig(file)
}
