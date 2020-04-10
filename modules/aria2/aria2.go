package aria2

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gregdel/argo/rpc"
	polochon "github.com/odwrtw/polochon/lib"
	yaml "gopkg.in/yaml.v2"
)

// Make sure that the module is a downloader
var _ polochon.Downloader = (*Client)(nil)

// Register a new Downloader
func init() {
	polochon.RegisterModule(&Client{})
}

// Module constants
const (
	moduleName = "aria2"
)

// Params represents the module params
type Params struct {
	URL    string `yaml:"url"`
	Secret string `yaml:"secret"`
}

// Client holds the connection with transmission
type Client struct {
	*Params
	protocol   rpc.Protocol
	configured bool
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

// InitWithParams helps init the module with the given params
func (c *Client) InitWithParams(params *Params) error {
	if params.URL == "" {
		return fmt.Errorf("aria2: missing URL")
	}

	if params.Secret == "" {
		return fmt.Errorf("aria2: missing rpc secret")
	}

	var err error
	c.protocol, err = rpc.New(params.URL, params.Secret)
	if err != nil {
		return err
	}

	c.configured = true

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
	if torrent.URL == "" {
		return fmt.Errorf("aria2: missing torrent URL")
	}

	_, err := c.protocol.AddURI(torrent.URL)
	return err
}

// List implements the downloader interface
func (c *Client) List() ([]*polochon.Torrent, error) {
	list := []rpc.StatusInfo{}

	// Get active downloads
	l, err := c.protocol.TellActive()
	if err != nil {
		return nil, err
	}
	if len(l) != 0 {
		list = append(list, l...)
	}

	// Get stopped and waiting downloads
	for _, f := range []func(int, int, ...string) ([]rpc.StatusInfo, error){
		c.protocol.TellWaiting,
		c.protocol.TellStopped,
	} {
		l, err := f(0, 10)
		if err != nil {
			return nil, err
		}

		if len(l) != 0 {
			list = append(list, l...)
		}
	}

	result := []*polochon.Torrent{}
	for _, status := range list {
		i := &polochon.Torrent{
			ID:   status.Gid,
			Name: status.BitTorrent.Info.Name,
		}

		// Add the filePaths
		i.FilePaths = []string{}
		for _, f := range status.Files {
			i.FilePaths = append(i.FilePaths, f.Path)
		}

		// Set the path as the default name
		if i.Name == "" && len(i.FilePaths) > 0 {
			i.Name = i.FilePaths[0]
		}

		for i, s := range map[*int]string{
			&i.DownloadRate:   status.DownloadSpeed,
			&i.UploadRate:     status.UploadSpeed,
			&i.DownloadedSize: status.CompletedLength,
			&i.UploadedSize:   status.UploadLength,
			&i.TotalSize:      status.TotalLength,
		} {
			var err error
			*i, err = strconv.Atoi(s)
			if err != nil {
				continue
			}
		}

		if status.CompletedLength == status.TotalLength {
			i.IsFinished = true
			i.PercentDone = 100
		} else {
			i.PercentDone = float32(i.DownloadedSize) * 100 / float32(i.TotalSize)
		}

		if i.UploadedSize != 0 {
			i.Ratio = float32(i.UploadedSize) / float32(i.TotalSize)
		}

		result = append(result, i)
	}

	return result, nil
}

// Remove implements the downloader interface
func (c *Client) Remove(torrent *polochon.Torrent) error {
	if torrent.ID == "" {
		return fmt.Errorf("aria2: no id to remove the download")
	}

	_, err := c.protocol.Remove(torrent.ID)
	if err != nil {
		if strings.Contains(err.Error(), "Active Download not found") {
			// This downloadable is not active
		} else {
			return err
		}
	}

	_, err = c.protocol.PurgeDownloadResult()
	return err
}
