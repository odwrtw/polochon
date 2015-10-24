package transmission

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/transmission"
)

// Module constants
const (
	moduleName = "transmission"
)

// Register a new Downloader
func init() {
	polochon.RegisterDownloader(moduleName, New)
}

// Client holds the connection with transmission
type Client struct {
	URL       string `yaml:"url"`
	CheckSSL  bool   `yaml:"check_ssl"`
	BasicAuth bool   `yaml:"basic_auth"`
	Username  string `yaml:"user"`
	Password  string `yaml:"password"`
	tClient   *transmission.Client
}

// New module
func New(p []byte) (polochon.Downloader, error) {
	client := &Client{}

	if err := yaml.Unmarshal(p, client); err != nil {
		return nil, err
	}

	if err := client.checkConfig(); err != nil {
		return nil, err
	}

	// Set the transmission client according to the conf
	if err := client.setTransmissionClient(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) checkConfig() error {
	if c.URL == "" {
		return fmt.Errorf("transmission: missing URL")
	}

	if c.BasicAuth {
		if c.Username == "" || c.Password == "" {
			return fmt.Errorf("transmission: missing authentication params")
		}
	}

	return nil
}

func (c *Client) setTransmissionClient() error {
	skipSSL := !c.CheckSSL

	// Create HTTP client with SSL configuration
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSSL},
	}
	httpClient := http.Client{Transport: tr}

	conf := transmission.Config{
		Address:    c.URL,
		User:       c.Username,
		Password:   c.Password,
		HTTPClient: &httpClient,
	}

	t, err := transmission.New(conf)
	if err != nil {
		return err
	}

	c.tClient = t

	return nil
}

// Name implements the Module interface
func (c *Client) Name() string {
	return moduleName
}

// Download implements the downloader interface
func (c *Client) Download(URL string, log *logrus.Entry) error {
	_, err := c.tClient.Add(URL)
	if err != nil {
		if err == transmission.ErrDuplicateTorrent {
			return polochon.ErrDuplicateTorrent
		}
		return err
	}
	return nil
}
