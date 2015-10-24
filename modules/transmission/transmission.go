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
	polochon.RegisterDownloader(moduleName, NewFromRawYaml)
}

// Params represents the module params
type Params struct {
	URL       string `yaml:"url"`
	CheckSSL  bool   `yaml:"check_ssl"`
	BasicAuth bool   `yaml:"basic_auth"`
	Username  string `yaml:"user"`
	Password  string `yaml:"password"`
}

// Client holds the connection with transmission
type Client struct {
	*Params
	tClient *transmission.Client
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Downloader, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New module
func New(params *Params) (polochon.Downloader, error) {
	client := &Client{Params: params}

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
