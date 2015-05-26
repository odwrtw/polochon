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

// Config represents polochon's config
type Config struct {
	Watcher       WatcherConfig       `yaml:"watcher"`
	Downloader    DownloaderConfig    `yaml:"downloader"`
	ModulesParams []map[string]string `yaml:"modules_params"`
	Video         VideoConfig         `yaml:"video"`
	Show          ShowConfig          `yaml:"show"`
	Movie         MovieConfig         `yaml:"movie"`
	Modules       *Modules            `yaml:"-"`
	Log           *logrus.Entry       `yaml:"-"`
}

// WatcherConfig represents the configuration for the detailers
type WatcherConfig struct {
	Timer          time.Duration `yaml:"timer"`
	Dir            string        `yaml:"dir"`
	FsNotifierName string        `yaml:"fsnotifier"`
	FsNotifier     FsNotifier    `yaml:"-"`
}

// VideoConfig represents the configuration for the detailers
type VideoConfig struct {
	GuesserName               string   `yaml:"guesser"`
	Guesser                   Guesser  `yaml:"-"`
	ExcludeFileContaining     []string `yaml:"exclude_file_containing"`
	VideoExtentions           []string `yaml:"allowed_file_extensions"`
	AllowedExtentionsToDelete []string `yaml:"allowed_file_extensions_to_delete"`
}

// ShowConfig represents the configuration for a show and its show episodes
type ShowConfig struct {
	Dir            string      `yaml:"dir"`
	TorrenterNames []string    `yaml:"torrenters"`
	Torrenters     []Torrenter `yaml:"-"`
	DetailerNames  []string    `yaml:"detailers"`
	Detailers      []Detailer  `yaml:"-"`
}

// MovieConfig represents the configuration for a movie
type MovieConfig struct {
	Dir            string      `yaml:"dir"`
	TorrenterNames []string    `yaml:"torrenters"`
	Torrenters     []Torrenter `yaml:"-"`
	DetailerNames  []string    `yaml:"detailers"`
	Detailers      []Detailer  `yaml:"-"`
}

// DownloaderConfig represents the configuration for the downloader
type DownloaderConfig struct {
	DownloadDir string `yaml:"download_dir"`
}

// readConfig helps read the config
func readConfig(r io.Reader) (*Config, error) {
	c := &Config{}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// writeConfig helps writes the config
func (c *Config) write(w io.Writer) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	if err != nil {
		return err
	}

	return nil

}

// Init the configuration
func (c *Config) Init() error {
	// Configure the modules
	if err := c.initLogger(); err != nil {
		return err
	}

	// Configure the modules
	if err := c.initModules(); err != nil {
		return err
	}

	// Init watcher
	if err := c.initWatcher(); err != nil {
		return err
	}

	// Init video
	if err := c.initVideo(); err != nil {
		return err
	}

	// Init movie
	if err := c.initMovie(); err != nil {
		return err
	}

	// Init show
	if err := c.initShow(); err != nil {
		return err
	}

	return nil
}

func (c *Config) initLogger() error {
	// Setup the logger
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	logger.Out = os.Stderr
	logger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}

	// Add it to the config
	c.Log = logrus.NewEntry(logger)

	return nil
}

func (c *Config) initModules() error {
	c.Modules = NewModules(c.Log)
	return nil
}

// moduleParams returuns the modules params set in the configuration.
func (c *Config) moduleParams(moduleName string) (map[string]string, error) {
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

func (c *Config) initWatcher() error {
	if c.Watcher.FsNotifierName == "" {
		return fmt.Errorf("config: missing watcher fsnotifier name")
	}

	// get params
	moduleParams, err := c.moduleParams(c.Watcher.FsNotifierName)
	if err != nil {
		return err
	}

	// Configure
	if err := c.Modules.ConfigureFsNotifier(c.Watcher.FsNotifierName, moduleParams); err != nil {
		return err
	}

	n, err := c.Modules.FsNotifier(c.Watcher.FsNotifierName)
	if err != nil {
		return err
	}
	c.Watcher.FsNotifier = n

	return nil
}

func (c *Config) initVideo() error {
	// Get video guesser
	if c.Video.GuesserName == "" {
		return fmt.Errorf("config: missing video guesser name")
	}

	// get params
	moduleParams, err := c.moduleParams(c.Video.GuesserName)
	if err != nil {
		return err
	}

	// Configure
	if err := c.Modules.ConfigureGuesser(c.Video.GuesserName, moduleParams); err != nil {
		return err
	}

	g, err := c.Modules.Guesser(c.Video.GuesserName)
	if err != nil {
		return err
	}
	c.Video.Guesser = g

	return nil
}

func (c *Config) initMovie() error {
	// Get movie detailer
	if len(c.Movie.DetailerNames) == 0 {
		return fmt.Errorf("config: missing movie detailer names")
	}

	for _, detailerName := range c.Movie.DetailerNames {
		moduleParams, err := c.moduleParams(detailerName)
		if err != nil {
			return err
		}

		if err := c.Modules.ConfigureDetailer(detailerName, moduleParams); err != nil {
			return err
		}

		d, err := c.Modules.Detailer(detailerName)
		if err != nil {
			return err
		}
		c.Movie.Detailers = append(c.Movie.Detailers, d)
	}

	// Get movie torrenter
	if len(c.Movie.TorrenterNames) == 0 {
		return fmt.Errorf("config: missing movie torrenter names")
	}

	for _, torrenterName := range c.Movie.TorrenterNames {
		moduleParams, err := c.moduleParams(torrenterName)
		if err != nil {
			return err
		}

		if err := c.Modules.ConfigureTorrenter(torrenterName, moduleParams); err != nil {
			return err
		}

		t, err := c.Modules.Torrenter(torrenterName)
		if err != nil {
			return err
		}

		c.Movie.Torrenters = append(c.Movie.Torrenters, t)
	}

	return nil
}

func (c *Config) initShow() error {
	// Get show detailer
	if len(c.Show.DetailerNames) == 0 {
		return fmt.Errorf("config: missing show detailer names")
	}

	for _, detailerName := range c.Show.DetailerNames {
		moduleParams, err := c.moduleParams(detailerName)
		if err != nil {
			return err
		}

		if err := c.Modules.ConfigureDetailer(detailerName, moduleParams); err != nil {
			return err
		}

		d, err := c.Modules.Detailer(detailerName)
		if err != nil {
			return err
		}
		c.Show.Detailers = append(c.Show.Detailers, d)
	}

	// Get show torrenter
	if len(c.Show.TorrenterNames) == 0 {
		return fmt.Errorf("config: missing movie torrenter names")
	}

	for _, torrenterName := range c.Show.TorrenterNames {
		moduleParams, err := c.moduleParams(torrenterName)
		if err != nil {
			return err
		}

		if err := c.Modules.ConfigureTorrenter(torrenterName, moduleParams); err != nil {
			return err
		}

		t, err := c.Modules.Torrenter(torrenterName)
		if err != nil {
			return err
		}

		c.Show.Torrenters = append(c.Show.Torrenters, t)
	}

	return nil
}

// ReadConfigFile reads a file from a path and returns a config
func ReadConfigFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return readConfig(file)
}

// WriteConfigFile writes the config into file
func (c *Config) WriteConfigFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return c.write(file)
}
