package transmission

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

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
	URL       string
	CheckSSL  bool
	BasicAuth bool
	Username  string
	Password  string
	tClient   *transmission.Client
	log       *logrus.Entry
}

// New module
func New(params map[string]interface{}, log *logrus.Entry) (polochon.Downloader, error) {
	var URL, username, password string

	// Check SSL and basic authentication by default
	checkSSL, basicAuth := true, true

	for ptr, param := range map[*bool]string{
		&checkSSL:  "check_ssl",
		&basicAuth: "basic_auth",
	} {
		p, ok := params[param]
		if !ok {
			continue
		}

		v, ok := p.(bool)
		if !ok {
			return nil, fmt.Errorf("transmission: %s should be a bool", param)
		}

		*ptr = v
	}

	for ptr, param := range map[*string]string{
		&URL:      "url",
		&username: "user",
		&password: "password",
	} {
		p, ok := params[param]
		if !ok {
			continue
		}

		v, ok := p.(string)
		if !ok {
			return nil, fmt.Errorf("transmission: %s should be a string", param)
		}

		*ptr = v
	}

	client := &Client{
		URL:       URL,
		CheckSSL:  checkSSL,
		BasicAuth: basicAuth,
		Username:  username,
		Password:  password,
		log:       log,
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
func (c *Client) Download(URL string) error {
	start := time.Now()
	_, err := c.tClient.Add(URL)
	if err != nil {
		return err
	}

	c.log.WithFields(logrus.Fields{
		"URL":       URL,
		"timeToAdd": time.Since(start),
	}).Debug("torrent added to transmission")

	return nil
}
