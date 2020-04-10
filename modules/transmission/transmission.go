package transmission

import (
	"crypto/tls"
	"errors"
	"net/http"
	"strconv"

	yaml "gopkg.in/yaml.v2"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/transmission"
)

// Make sure that the module is a downloader
var _ polochon.Downloader = (*Client)(nil)

// Custom errors
var (
	ErrMissingServerURL         = errors.New("transmission: missing server URL")
	ErrMissingServerCredentials = errors.New("transmission: missing server credentials")
	ErrMissingURL               = errors.New("transmission: missing torrent URL")
	ErrMissingID                = errors.New("transmission: missing torrent ID")
	ErrInvalidID                = errors.New("transmission: invalid ID, not an int")
)

func init() {
	polochon.RegisterModule(&Client{})
}

// Module constants
const (
	moduleName = "transmission"
)

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
	transmission *transmission.Client
	configured   bool
}

// Init implements the module interface
func (c *Client) Init(p []byte) error {
	if c.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return c.InitWithParams(params)
}

// InitWithParams configures the module
func (c *Client) InitWithParams(params *Params) error {
	c.Params = params
	if err := c.checkConfig(); err != nil {
		return err
	}

	// Set the transmission client according to the conf
	if err := c.setTransmissionClient(); err != nil {
		return err
	}

	c.configured = true

	return nil
}

func (c *Client) checkConfig() error {
	if c.URL == "" {
		return ErrMissingServerURL
	}

	if c.BasicAuth {
		if c.Username == "" || c.Password == "" {
			return ErrMissingServerCredentials
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
		HTTPClient: &httpClient}

	t, err := transmission.New(conf)
	if err != nil {
		return err
	}

	c.transmission = t

	return nil
}

// Name implements the Module interface
func (c *Client) Name() string {
	return moduleName
}

// Status implements the Module interface
func (c *Client) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// Download implements the downloader interface
func (c *Client) Download(torrent *polochon.Torrent) error {
	if torrent.Result == nil || torrent.Result.URL == "" {
		return ErrMissingURL
	}

	t, err := c.transmission.Add(torrent.Result.URL)
	if err != nil {
		return err
	}

	labels := labels(torrent)
	if labels == nil {
		return nil
	}

	return t.Set(transmission.SetTorrentArg{
		Labels: labels,
	})
}

// List implements the downloader interface
func (c *Client) List() ([]*polochon.Torrent, error) {
	tt, err := c.transmission.GetTorrents()
	if err != nil {
		return nil, err
	}

	var torrents []*polochon.Torrent
	for _, t := range tt {

		isFinished := false

		// Check that the torrent is finished
		if t.PercentDone == 1 {
			isFinished = true
		}

		// Add the filePaths
		var filePaths []string
		if t.Files != nil {
			for _, f := range *t.Files {
				filePaths = append(filePaths, f.Name)
			}
		}

		torrent := &polochon.Torrent{
			Status: &polochon.TorrentStatus{
				ID:             strconv.Itoa(t.ID),
				DownloadRate:   t.RateDownload,
				DownloadedSize: int(t.DownloadedEver),
				UploadedSize:   int(t.UploadedEver),
				FilePaths:      filePaths,
				IsFinished:     isFinished,
				Name:           t.Name,
				PercentDone:    float32(t.PercentDone) * 100,
				Ratio:          float32(t.UploadRatio),
				TotalSize:      int(t.SizeWhenDone),
				UploadRate:     t.RateUpload,
			},
		}
		updateFromLabel(torrent, t.Labels)

		torrents = append(torrents, torrent)
	}

	return torrents, nil
}

// Remove implements the downloader interface
func (c *Client) Remove(torrent *polochon.Torrent) error {
	if torrent.Status == nil || torrent.Status.ID == "" {
		return ErrMissingID
	}

	id, err := strconv.Atoi(torrent.Status.ID)
	if err != nil {
		return ErrInvalidID
	}

	// Delete the torrent and the data
	return c.transmission.RemoveTorrents(
		[]*transmission.Torrent{{ID: id}}, false)
}
